#!/bin/sh
#
# Copyright 2017 The HAProxy Ingress Controller Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

if [ $# -gt 0 ] && [ "$(echo $1 | cut -b1-2)" != "--" ]; then
    # Probably a `docker run -ti`, so exec and exit
    exec "$@"
elif [ "$1" = "--init" ]; then
    port="${2:-10253}"
    cat >/etc/haproxy/haproxy.cfg <<EOF
defaults
    timeout server 1s
    timeout client 1s
    timeout connect 1s
EOF
else
    # Copy static files to /etc/haproxy, which cannot have static content
    cp -R /etc/lua /etc/haproxy/ 
    exec /haproxy-ingress-controller "$@"
fi
