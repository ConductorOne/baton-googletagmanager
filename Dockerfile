FROM gcr.io/distroless/static-debian11:nonroot
ENTRYPOINT ["/baton-googletagmanager"]
COPY baton-googletagmanager /