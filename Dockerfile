FROM golang:1.26 AS build

ARG HTTP_FS_REF=f8b33d1c2bb79f78e95cf219358224fada94de80
WORKDIR /src
RUN curl --fail --location --silent --show-error \
    "https://github.com/zaker/http-fs/archive/${HTTP_FS_REF}.tar.gz" -o /tmp/source.tar.gz \
    && tar -xzf /tmp/source.tar.gz --strip-components=1 -C /src \
    && rm -rf /tmp/source.tar.gz /src/static
COPY static/ ./static/
RUN CGO_ENABLED=0 go build -trimpath -o /http-fs .

FROM scratch
COPY --from=build /http-fs /http-fs
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/http-fs"]
