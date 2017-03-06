# trago

Trago is a file synchronization utility written in Go.

Inspired by [tra](https://swtch.com/tra/).

This is largely a work in progress. Right now, we create a remote
instance of `trago` and these instances communicate through their
respective stdout/stdin.  The original `tra` creates a central
master process (`tra`) and two identical `trasrv` processes. The
`tra` process then handles the communication between the `trasrv`s
through RPC.

I hope to write that version someday, once I make this one a little
less messy.
