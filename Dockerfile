FROM gcr.io/distroless/static-debian11:nonroot
ENTRYPOINT ["/baton-expensify"]
COPY baton-expensify /