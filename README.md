# Professor

Worker for the [Hack4Impact-UMD App Portal](apply.umd.hack4impact.org) assessment autograder.

## Tech Stack

- **Language**: Go
- **Runtime**: Bun
- **Testing**: Playwright
- **Database**: Firestore
- **Deployment**: Docker, Google Cloud Run

## Local Development

Build locally with:

```bash
make build
```

The resulting binary will be `professor`.

With Docker installed and alive, build the image with:

```bash
make docker-build
```

Then run with:

```bash
make docker-run
```

And now `http://localhost::8080` on your machine should be live!
