# trago

Trago is a file synchronization utility inspired by
[tra](https://swtch.com/tra/).

This is largely a work in progress. Right now, our local `trago`
process spawns a remote process through ssh, and these instances
communicate through their respective stdout/stdin.  The original `tra`
creates a central master process (`tra`) and two identical `trasrv`
processes. The `tra` process then handles the communication between
the `trasrv`s through RPC.

I hope to write that version someday, once I make this one a little
less messy.

UPDATE: The RPC version seems to be coming along nicely! The old text-based
protocol has been replaced.
