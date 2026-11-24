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
