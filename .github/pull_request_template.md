## 📋 Description

Briefly describe what this PR does and why it's needed.

**Related Issue:** #[ISSUE_NUMBER]

## 🚀 Testing Environment

To trigger a fresh deployment of all services in the PR preview environment, comment:
```
update deploy
```

**Automatic PR environments:**
- Backend: https://my-backend-qa-pr-[PR_NUMBER].onrender.com
- Collect: https://my-collect-qa-pr-[PR_NUMBER].onrender.com
- Frontend: https://my-frontend-qa-pr-[PR_NUMBER].onrender.com
- Proxy: https://my-proxy-qa-pr-[PR_NUMBER].onrender.com

## ✅ Merge Checklist

**Code Quality:**
- [![Backend](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci-main.yml?job=backend-tests&label=Backend%20Tests&branch=[PR_BRANCH])](https://github.com/NethServer/my/actions/workflows/ci-main.yml)
- [![Collect](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci-main.yml?job=collect-tests&label=Collect%20Tests&branch=[PR_BRANCH])](https://github.com/NethServer/my/actions/workflows/ci-main.yml)
- [![Sync](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci-main.yml?job=sync-tests&label=Sync%20Tests&branch=[PR_BRANCH])](https://github.com/NethServer/my/actions/workflows/ci-main.yml)
- [![Frontend](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci-main.yml?job=frontend-tests&label=Frontend%20Tests&branch=[PR_BRANCH])](https://github.com/NethServer/my/actions/workflows/ci-main.yml)

**Builds:**
- [![Backend](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci-main.yml?job=backend-build&label=Backend&branch=[PR_BRANCH])](https://github.com/NethServer/my/actions/workflows/ci-main.yml)
- [![Collect](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci-main.yml?job=collect-build&label=Collect&branch=[PR_BRANCH])](https://github.com/NethServer/my/actions/workflows/ci-main.yml)
- [![Sync](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci-main.yml?job=sync-build&label=Sync&branch=[PR_BRANCH])](https://github.com/NethServer/my/actions/workflows/ci-main.yml)
- [![Frontend](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci-main.yml?job=frontend-build&label=Frontend&branch=[PR_BRANCH])](https://github.com/NethServer/my/actions/workflows/ci-main.yml)
