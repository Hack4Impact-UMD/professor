# Professor

Worker for the [Hack4Impact-UMD App Portal](apply.umd.hack4impact.org) assessment autograder.

## Local Development

With Docker installed and alive, build with:

```bash
docker build -t professor .
```

Then run with:

```
docker run --rm -p 8080:8080 -e PORT=8080 professor
```

And now `http://localhost::8080` on your machine should be live!
