FROM redhat/ubi8-minimal
RUN microdnf update && microdnf install findutils procps tar gzip systemd && microdnf clean all

COPY ./*.rpm .
RUN rpm -i "*rpm"
COPY tests /config/tests
