/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package status

import (
    "context"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	pool "gopkg.in/go-playground/pool.v3"
	apiv1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"

	"github.com/jcmoraisjr/haproxy-ingress/pkg/common/ingress/annotations/class"
	"github.com/jcmoraisjr/haproxy-ingress/pkg/common/ingress/store"
	"github.com/jcmoraisjr/haproxy-ingress/pkg/common/k8s"
	"github.com/jcmoraisjr/haproxy-ingress/pkg/common/task"
	"github.com/jcmoraisjr/haproxy-ingress/pkg/common/utils"
)

const (
	updateInterval = 60 * time.Second
)

// Sync ...
type Sync interface {
	Run(stopCh <-chan struct{})
	Shutdown()
}

// Config ...
type Config struct {
	Client clientset.Interface

	PublishService string

	ElectionID string

	UpdateStatusOnShutdown bool

	UseNodeInternalIP bool

	IngressLister store.IngressLister

	DefaultIngressClass string
	IngressClass        string

	// CustomIngressStatus allows to set custom values in Ingress status
	CustomIngressStatus func(*networking.Ingress) []apiv1.LoadBalancerIngress
}

// statusSync keeps the status IP in each Ingress rule updated executing a periodic check
// in all the defined rules. To simplify the process leader election is used so the update
// is executed only in one node (Ingress controllers can be scaled to more than one)
// If the controller is running with the flag --publish-service (with a valid service)
// the IP address behind the service is used, if not the source is the IP/s of the node/s
type statusSync struct {
	Config
	// pod contains runtime information about this pod
	pod *k8s.PodInfo

	elector *leaderelection.LeaderElector
	// workqueue used to keep in sync the status IP/s
	// in the Ingress rules
	syncQueue *task.Queue
}

// Run starts the loop to keep the status in sync
func (s statusSync) Run(stopCh <-chan struct{}) {
    ctx := context.Background()
	go s.elector.Run(ctx)
	go wait.Forever(s.update, updateInterval)
	go s.syncQueue.Run(time.Second, stopCh)
	<-stopCh
}

func (s *statusSync) update() {
	// send a dummy object to the queue to force a sync
	s.syncQueue.Enqueue("sync status")
}

// Shutdown stop the sync. In case the instance is the leader it will remove the current IP
// if there is no other instances running.
func (s statusSync) Shutdown() {
	go s.syncQueue.Shutdown()
	// remove IP from Ingress
	if !s.elector.IsLeader() {
		return
	}

	if !s.UpdateStatusOnShutdown {
		glog.Warningf("skipping update of status of Ingress rules")
		return
	}

	glog.Infof("updating status of Ingress rules (remove)")

	addrs, err := s.runningAddresses()
	if err != nil {
		glog.Errorf("error obtaining running IPs: %v", addrs)
		return
	}

	if len(addrs) > 1 {
		// leave the job to the next leader
		glog.Infof("leaving status update for next leader (%v)", len(addrs))
		return
	}

	if s.isRunningMultiplePods() {
		glog.V(2).Infof("skipping Ingress status update (multiple pods running - another one will be elected as master)")
		return
	}

	glog.Infof("removing address from ingress status (%v)", addrs)
	s.updateStatus([]apiv1.LoadBalancerIngress{})
}

func (s *statusSync) sync(key interface{}) error {
	if s.syncQueue.IsShuttingDown() {
		glog.V(2).Infof("skipping Ingress status update (shutting down in progress)")
		return nil
	}

	if !s.elector.IsLeader() {
		glog.V(2).Infof("skipping Ingress status update (I am not the current leader)")
		return nil
	}

	addrs, err := s.runningAddresses()
	if err != nil {
		return err
	}
	s.updateStatus(sliceToStatus(addrs))

	return nil
}

func (s statusSync) keyfunc(input interface{}) (interface{}, error) {
	return input, nil
}

// NewStatusSyncer returns a new Sync instance
func NewStatusSyncer(config Config) Sync {
	pod, err := k8s.GetPodDetails(config.Client)
	if err != nil {
		glog.Fatalf("unexpected error obtaining pod information: %v", err)
	}

	st := statusSync{
		pod: pod,

		Config: config,
	}
	st.syncQueue = task.NewCustomTaskQueue(st.sync, st.keyfunc)

	// we need to use the defined ingress class to allow multiple leaders
	// in order to update information about ingress status
	electionID := fmt.Sprintf("%v-%v", config.ElectionID, config.DefaultIngressClass)
	if config.IngressClass != "" {
		electionID = fmt.Sprintf("%v-%v", config.ElectionID, config.IngressClass)
	}

	callbacks := leaderelection.LeaderCallbacks{
		OnStartedLeading: func(context.Context) {
			glog.V(2).Infof("I am the new status update leader")
		},
		OnStoppedLeading: func() {
			glog.V(2).Infof("I am not status update leader anymore")
		},
		OnNewLeader: func(identity string) {
			glog.Infof("new leader elected: %v", identity)
		},
	}

	broadcaster := record.NewBroadcaster()
	hostname, _ := os.Hostname()

	recorder := broadcaster.NewRecorder(scheme.Scheme, apiv1.EventSource{
		Component: "ingress-leader-elector",
		Host:      hostname,
	})

	lock := resourcelock.ConfigMapLock{
		ConfigMapMeta: metav1.ObjectMeta{Namespace: pod.Namespace, Name: electionID},
		Client:        config.Client.CoreV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity:      pod.Name,
			EventRecorder: recorder,
		},
	}

	ttl := 30 * time.Second
	le, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:          &lock,
		LeaseDuration: ttl,
		RenewDeadline: ttl / 2,
		RetryPeriod:   ttl / 4,
		Callbacks:     callbacks,
	})

	if err != nil {
		glog.Fatalf("unexpected error starting leader election: %v", err)
	}

	st.elector = le
	return st
}

// runningAddresses returns a list of IP addresses and/or FQDN where the
// ingress controller is currently running
func (s *statusSync) runningAddresses() ([]string, error) {
    ctx := context.Background()
	if s.PublishService != "" {
		ns, name, _ := k8s.ParseNameNS(s.PublishService)
		svc, err := s.Client.CoreV1().Services(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		addrs := []string{}
		for _, ip := range svc.Status.LoadBalancer.Ingress {
			if ip.IP == "" {
				addrs = append(addrs, ip.Hostname)
			} else {
				addrs = append(addrs, ip.IP)
			}
		}
		for _, ip := range svc.Spec.ExternalIPs {
			addrs = append(addrs, ip)
		}

		return addrs, nil
	}

	// get information about all the pods running the ingress controller
	pods, err := s.Client.CoreV1().Pods(s.pod.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(s.pod.Labels).String(),
	})
	if err != nil {
		return nil, err
	}

	addrs := []string{}
	for _, pod := range pods.Items {
		name := k8s.GetNodeIP(s.Client, pod.Spec.NodeName, s.UseNodeInternalIP)
		if !utils.StringInSlice(name, addrs) {
			addrs = append(addrs, name)
		}
	}
	return addrs, nil
}

func (s *statusSync) isRunningMultiplePods() bool {
    ctx := context.Background()
	pods, err := s.Client.CoreV1().Pods(s.pod.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(s.pod.Labels).String(),
	})
	if err != nil {
		return false
	}

	return len(pods.Items) > 1
}

// sliceToStatus converts a slice of IP and/or hostnames to LoadBalancerIngress
func sliceToStatus(endpoints []string) []apiv1.LoadBalancerIngress {
	lbi := []apiv1.LoadBalancerIngress{}
	for _, ep := range endpoints {
		if net.ParseIP(ep) == nil {
			lbi = append(lbi, apiv1.LoadBalancerIngress{Hostname: ep})
		} else {
			lbi = append(lbi, apiv1.LoadBalancerIngress{IP: ep})
		}
	}

	sort.SliceStable(lbi, func(a, b int) bool {
		return lbi[a].IP < lbi[b].IP
	})

	return lbi
}

// updateStatus changes the status information of Ingress rules
// If the backend function CustomIngressStatus returns a value different
// of nil then it uses the returned value or the newIngressPoint values
func (s *statusSync) updateStatus(newIngressPoint []apiv1.LoadBalancerIngress) {
	ings := s.IngressLister.List()

	p := pool.NewLimited(10)
	defer p.Close()

	batch := p.Batch()

	for _, cur := range ings {
		ing := cur.(*networking.Ingress)

		if !class.IsValid(ing, s.Config.IngressClass, s.Config.DefaultIngressClass) {
			continue
		}

		batch.Queue(runUpdate(ing, newIngressPoint, s.Client, s.CustomIngressStatus))
	}

	batch.QueueComplete()
	batch.WaitAll()
}

func runUpdate(ing *networking.Ingress, status []apiv1.LoadBalancerIngress,
	client clientset.Interface,
	statusFunc func(*networking.Ingress) []apiv1.LoadBalancerIngress) pool.WorkFunc {
    ctx := context.Background()
	return func(wu pool.WorkUnit) (interface{}, error) {
		if wu.IsCancelled() {
			return nil, nil
		}

		addrs := status
		ca := statusFunc(ing)
		if ca != nil {
			addrs = ca
		}
		sort.SliceStable(addrs, lessLoadBalancerIngress(addrs))

		curIPs := ing.Status.LoadBalancer.Ingress
		sort.SliceStable(curIPs, lessLoadBalancerIngress(curIPs))

		if ingressSliceEqual(addrs, curIPs) {
			glog.V(3).Infof("skipping update of Ingress %v/%v (no change)", ing.Namespace, ing.Name)
			return true, nil
		}

		ingClient := client.NetworkingV1().Ingresses(ing.Namespace)

		currIng, err := ingClient.Get(ctx, ing.Name, metav1.GetOptions{})
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("unexpected error searching Ingress %v/%v", ing.Namespace, ing.Name))
		}

		glog.Infof("updating Ingress %v/%v status to %v", currIng.Namespace, currIng.Name, addrs)
		currIng.Status.LoadBalancer.Ingress = addrs
		_, err = ingClient.UpdateStatus(ctx, currIng, metav1.UpdateOptions{})
		if err != nil {
			glog.Warningf("error updating ingress rule: %v", err)
		}

		return true, nil
	}
}

func lessLoadBalancerIngress(addrs []apiv1.LoadBalancerIngress) func(int, int) bool {
	return func(a, b int) bool {
		switch strings.Compare(addrs[a].Hostname, addrs[b].Hostname) {
		case -1:
			return true
		case 1:
			return false
		}
		return addrs[a].IP < addrs[b].IP
	}
}

func ingressSliceEqual(lhs, rhs []apiv1.LoadBalancerIngress) bool {
	if len(lhs) != len(rhs) {
		return false
	}

	for i := range lhs {
		if lhs[i].IP != rhs[i].IP {
			return false
		}
		if lhs[i].Hostname != rhs[i].Hostname {
			return false
		}
	}
	return true
}
