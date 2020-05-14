FROM registry.access.redhat.com/ubi8/ubi-minimal:8.1-398

LABEL   name="quay.io/hybridappio/ham-deployable-operator" \
        vendor="IBM" \
        summary="Hybrid Application Model - Deployable Operator" \
        description="Handles deployable components running on non-kubernetes platform(s)."


ENV OPERATOR=/usr/local/bin/ham-deployable-operator \
    USER_UID=1001 \
    USER_NAME=ham-deployable-operator

# install operator binary
COPY build/_output/bin/ham-deployable-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

# copy licenses
RUN mkdir /licenses
COPY LICENSE /licenses


ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
