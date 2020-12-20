FROM alpine:3.12

# Create Non Privilaged user
RUN addgroup --gid 101 anandnilkal && \
    adduser -S --uid 101 --ingroup anandnilkal anandnilkal

# Run as Non Privilaged user
USER anandnilkal

COPY bin/linux/aws-services-stream /go/bin/aws-services-stream
ENTRYPOINT /go/bin/aws-services-stream
