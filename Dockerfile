#
# Copyright 2021 The Sigstore Authors.
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

FROM golang:1.19.3@sha256:7ffa70183b7596e6bc1b78c132dbba9a6e05a26cd30eaa9832fecad64b83f029 as builder
ENV APP_ROOT=/opt/app-root
ENV GOPATH=$APP_ROOT

WORKDIR $APP_ROOT/src/
ADD go.mod go.sum $APP_ROOT/src/
RUN go mod download

# Add source code
ADD ./cmd/ $APP_ROOT/src/cmd/
ADD ./pkg/ $APP_ROOT/src/pkg/

RUN go build ./cmd/mirroring
RUN CGO_ENABLED=0 go build -gcflags "all=-N -l"  -o mirroring_debug ./cmd/mirroring

# Multi-Stage build
FROM golang:1.19.3@sha256:7ffa70183b7596e6bc1b78c132dbba9a6e05a26cd30eaa9832fecad64b83f029 as deploy

# Retrieve the binary from the previous stage
COPY --from=builder /opt/app-root/src/mirroring /usr/local/bin/mirroring

# Set the binary as the entrypoint of the container
CMD ["mirroring"]

# debug compile options & debugger
FROM deploy as debug
RUN go install github.com/go-delve/delve/cmd/dlv@v1.9.1

# overwrite utility and include debugger
COPY --from=builder /opt/app-root/src/mirroring_debug /usr/local/bin/mirroring
