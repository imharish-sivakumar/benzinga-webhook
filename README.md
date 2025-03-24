# 📦 Benzinga Webhook Receiver

A lightweight Go-based webhook receiver to handle and batch log entries with validation, batching logic, and configurable deployment infrastructure. Built for production-ready systems with CI/CD, infrastructure-as-code (Terraform + Ansible), and Dockerized deployment.

[![Go Webhook CI/CD](https://github.com/imharish-sivakumar/benzinga-webhook/actions/workflows/ci.yml/badge.svg)](https://github.com/imharish-sivakumar/benzinga-webhook/actions/workflows/ci.yml)

---

## ✨ Features

- 🔒 Input validation with `go-playground/validator`
- 📤 Batching of logs using either **batch size** or **interval**
- 📦 JSON-based logging format
- ✅ Health check endpoint
- 🧪 CI/CD using GitHub Actions
- 🚀 Deployed to EC2 (via Terraform + Ansible + Docker)

---

## 📂 Project Structure

```bash
.
├── Dockerfile
├── cmd
│   ├── main.go
│   └── main_test.go
├── docker-compose.yaml
├── go.mod / go.sum
├── infrastructure
│   ├── ansible
│   │   ├── inventory.ini
│   │   └── setup_nginx.yaml
│   ├── main.tf
│   └── variables.tf
└── internal
    ├── apperror
    ├── batcher
    ├── config
    ├── handler
    ├── logger
    └── model
```

---

## 🚀 Deployment Details

This application is deployed on **EC2** under `https://interviewwithhariharan.com`. It uses:

- **Terraform**: to provision EC2, security groups, Route53, ACM, and ALB.
- **Ansible**: to configure Docker, Docker Compose, and Nginx reverse proxy.

---

## 📮 API Endpoints

### `GET /healthz`
Returns a simple `200 OK` with `OK` body for health check.

### `POST /log`
Receives a single log payload (validated) and adds to the batch. If the batch size (5) is reached, it is sent to the `PostEndpoint`.

#### Sample Payload:
```json
{
   "user_id": 1,
   "total": 99.99,
   "title": "Example Log",
   "meta": {
      "logins": [{
         "time": "2020-08-08T01:52:50Z",
         "ip": "127.0.0.1"
      }],
      "phone_numbers": {
         "home": "123-4567-891",
         "mobile": "765-4321-912"
      }
   },
   "completed": true
}
```

---

## 🔧 Configuration (via ENV or `internal/config`)

| Variable         | Description                      | Default                                                     |
|------------------|----------------------------------|-------------------------------------------------------------|
| `BATCH_SIZE`     | Max number of logs in batch      | `5`                                                         |
| `BATCH_INTERVAL` | Time interval for batch flush    | `10s`                                                       |
| `POST_ENDPOINT`  | Target endpoint to send the logs | `https://webhook.site/0e9761c0-3966-45e2-b1dc-d675cb8752b4` |

---

## 🔄 Batch Trigger

When 5 logs are sent to `/log`, the batcher will POST them to:

🔗 `https://webhook.site/0e9761c0-3966-45e2-b1dc-d675cb8752b4`

### 🕵️‍♂️ To verify:
1. Send 5 valid POST requests to `/log`
2. Then, view the batched request at:
   - 👉 https://webhook.site/#!/view/0e9761c0-3966-45e2-b1dc-d675cb8752b4/bad89fd4-b724-4f4a-b01d-04f439555cf6/1

---

## 🧪 CI/CD

GitHub Actions runs the following jobs:
- ✅ `lint`: Linting via `golangci-lint`
- ✅ `test`: Unit testing + coverage enforcement (85%)
- ✅ `semgrep`: SAST security scan
- ✅ `docker`: Image build & push (DockerHub)

---

## 🐳 Docker

### Build Image:
```bash
docker build -t webhook-receiver .
```

### Run Locally:
```bash
docker run -p 8080:8080 webhook-receiver
```

---

## 🔐 Terraform & Ansible

Terraform provisions the infrastructure:
- EC2
- Route53 DNS
- ACM (HTTPS)
- ALB (Load Balancer)

Ansible sets up:
- Docker
- Docker Compose
- Nginx reverse proxy

---

## 👤 Author
**Deployed by:** [Hariharan Sivakumar](https://interviewwithhariharan.com)

---

## 📜 License
MIT License

---

Feel free to ⭐ this repo if you find it useful!