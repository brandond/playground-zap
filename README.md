#### zap logging playground

Zap supports passing `stdout` and `stderr` as output paths; these just wrap golang's default `os.Stdout` and `os.Stderr` variables. These outputs do not enforce a write deadline, and the underlying file descriptors can easily be put into blocking mode which prevents other callers from enforcing write deadlines.
Ref: https://github.com/golang/go/issues/24331 and https://github.com/uber-go/zap/issues/1196.

This means that all calls to a zap logger that output to `stdout` or `stderr` (along with all other file-based outputs) will **block the calling goroutine** when the write blocks. An example of this behavior is when a process is operating under containerd, which intentionally leaves container processes running when containerd is stopped - but ceases reading from the stdout/stderr pipes.

##### What's Here

This repo contains a demonstration zap log sink that supports non-blocking, deadlined IO against stdout, stderr, and any other file path. It also uses `syscall.Dup` to clone the `stdout` and `stderr` file descriptors, to ensure that no other user can make them blocking in the future. Note that this means that these loggers are **lossy** and may drop messages if the output pipe is full.

Since this is just a **DEMONSTRATION**, the log sink also writes a trace log to `nonblock.log` in the working directory, which prints debug info on writes. This can be used to validate that log writes are not blocking.

##### How To Use It

You can run this as: `kubectl run zap --image=docker.io/brandond/playground-zap:latest`. On K3s or RKE2 you can then stop the k3s/rke2 service (which also stops the container runtime) and then run `find /var/lib/rancher -name nonblock.log | xargs tail -f`. At some point you should start seeing: `Wrote 0 of 105: write /dev/stdout: i/o timeout`
