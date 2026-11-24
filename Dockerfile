FROM golang:1.26 AS build

WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /http-fs .

FROM scratch
COPY --from=build /http-fs /http-fs
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/http-fs"]
