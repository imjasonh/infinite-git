# Infinite Git

A Git HTTP server that generates a new commit every time someone pulls from the repository.

## Try it out

Clone the repo

```sh
git clone https://infinite-git-nd2dq3gc7a-uk.a.run.app/ /tmp/infinite-git && cd /tmp/infinite-git
```

Pull, and just keep pulling:

```
$ git pull
warning: no common commits
Unpacking objects: 100% (40/40), 4.14 KiB | 847.00 KiB/s, done.
From https://infinite-git-nd2dq3gc7a-uk.a.run.app
   df979d4..d483466  main       -> origin/main
Updating df979d4..d483466
Fast-forward
 hello.txt | 4 ++--
 1 file changed, 2 insertions(+), 2 deletions(-)
$ git pull
warning: no common commits
Unpacking objects: 100% (43/43), 4.45 KiB | 912.00 KiB/s, done.
From https://infinite-git-nd2dq3gc7a-uk.a.run.app
   d483466..47f0fcb  main       -> origin/main
Updating d483466..47f0fcb
Fast-forward
 hello.txt | 4 ++--
 1 file changed, 2 insertions(+), 2 deletions(-)
```

.....and so on.

## How It Works

1. When a client initiates a pull/clone, the server intercepts the reference discovery request
2. Before advertising refs, it generates a new commit with:
   - A unique file containing the pull counter and timestamp
   - A commit message indicating when the pull occurred
3. The new commit is added to the main branch
4. The updated refs are sent to the client
5. The client receives the new commit as part of the normal Git protocol flow

## Why?

I think it's neat!
