# http-fs

A dependency-free Go HTTP file server that serves only files embedded at build time.

Put your site in `static/`, then build and run the server:

```sh
go build -o http-fs .
./http-fs
```

By default it listens on `:8080`. Use `-addr` to change the listen address,
for example `./http-fs -addr 127.0.0.1:8000`.

The binary contains the files from `static/` and does not read them from disk
at runtime. Rebuild after changing the files. The directory must contain at
least one embeddable file; Go's `embed` excludes files and directories whose
names begin with `.` or `_`. See the [embed documentation](https://pkg.go.dev/embed)
for details.

## Container

Build the image after adding your files to `static/` (at least one embeddable
file is required):

```sh
docker build -t http-fs .
docker run --rm -p 8080:8080 http-fs
```

The build downloads the server source from a pinned GitHub archive and embeds
the local `static/` directory. Local Go source changes are not included. To
use a different GitHub commit, tag, or branch, pass
`--build-arg HTTP_FS_REF=<ref>` to `docker build`. Use a commit SHA for
reproducible builds; builds require access to GitHub.

The image uses a statically linked binary in a `scratch` container and runs as
an unprivileged user. To change the listen address, pass `-addr` after the
image name and publish the corresponding port. For external access, listen on
all interfaces rather than `127.0.0.1`.
