# trago

Trago is a file synchronization utility inspired by
[tra](https://swtch.com/tra/). It uses the synchronization algorithm
defined in the [tra paper](http://publications.csail.mit.edu/tmp/MIT-CSAIL-TR-2005-014.pdf).

## Usage

Simply point it to a remote directory and a local directory, and
trago will carry out a bidirectional sync.

```
    $ trago user@host:directory-A directory-B
```

## Caveats

The design is simple, and borders on primitive in some areas:

- moves and renames aren't detected
- uses ssh to start the remote process
- no per-directory worker threads
- uses a fileserver to download files
- files are transferred in their entirety
- conflicting files are skipped
