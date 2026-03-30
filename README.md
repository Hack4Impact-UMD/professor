# Professor

Worker for the [Hack4Impact-UMD App Portal](apply.umd.hack4impact.org) assessment autograder.

## Architecture

```mermaid
graph TB
    User[Applicant] -->|Submits Assessment| Frontend[App Portal - Frontend]
    Frontend[App Portal - Frontend] -->|Reads Status and Results| User[Applicant]
    Frontend -->|Requests Grading| Backend[App Portal - Backend]

    Backend -->|Creates Documents| Firestore[(Firestore)]
    Backend -->|Enqueues Job| CloudTasks[Queue - Cloud Tasks]

    CloudTasks -->|Delivers Job| Professor[Professor - Cloud Run]

    Professor -->|Updates Documents| Firestore
    Professor -->|Clones Repo| GitHub[GitHub]
    Professor -->|Tests Repo| Playwright[Playwright]

    Firestore -->|Reads Documents| Frontend

    style Frontend fill:#4285f4,color:#fff
    style Backend fill:#4285f4,color:#fff
    style Professor fill:#9334e9,color:#fff
    style Firestore fill:#ffa000,color:#fff
    style CloudTasks fill:#34a853,color:#fff
```

## Tech Stack

- **Language**: Go
- **Runtime**: Node
- **Package Manager**: Bun
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

**Make sure you have a .env file in the root directory with `PROJECT_ID` set correctly, and `DEV=true` in order to connect to the firestore emulator**

And now `http://localhost:8000` on your machine should be live!
