FROM gcr.io/distroless/static:nonroot
ARG BINARY TARGETARCH
COPY ${BINARY}-${TARGETARCH} /manager
ENTRYPOINT [ "/manager" ]
