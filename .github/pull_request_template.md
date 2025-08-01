## 📋 Description

Briefly describe what this PR does and why it's needed.

**Related Issue:** #[ISSUE_NUMBER]

## 🔧 Changes

- [ ] Backend
- [ ] Collect
- [ ] Frontend
- [ ] Proxy
- [ ] Sync
- [ ] Documentation
- [ ] Tests
- [ ] Build/Deploy

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

## ✅ Test Checklist

**Code Quality:**
- [ ] Backend: `make pre-commit` passes
- [ ] Collect: `make pre-commit` passes
- [ ] Sync: `make pre-commit` passes
- [ ] Frontend: `npm run pre-commit` passes

**Health Checks:**
- [ ] Backend: https://my-backend-qa-pr-[PR_NUMBER].onrender.com/api/health
- [ ] Collect: https://my-collect-qa-pr-[PR_NUMBER].onrender.com/api/health
- [ ] Frontend: https://my-frontend-qa-pr-[PR_NUMBER].onrender.com
- [ ] Proxy: https://my-proxy-qa-pr-[PR_NUMBER].onrender.com/health
