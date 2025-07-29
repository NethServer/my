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

**Automatic PR environments:**
- Backend: https://my-backend-qa-pr-[PR_NUMBER].onrender.com
- Collect: https://my-collect-qa-pr-[PR_NUMBER].onrender.com
- Frontend: https://my-frontend-qa-pr-[PR_NUMBER].onrender.com

**For Proxy service:**
```bash
cd proxy
perl -i -pe "s/LAST_UPDATE=.*/LAST_UPDATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)/" .render-build-trigger
git add .render-build-trigger
git commit -m "chore: trigger proxy rebuild"
git push
```

After build: https://my-proxy-qa-pr-[PR_NUMBER].onrender.com

## ✅ Test Checklist

**Code Quality:**
- [ ] Backend: `make pre-commit` passes
- [ ] Collect: `make pre-commit` passes
- [ ] Sync: `make pre-commit` passes
- [ ] Frontend: `make pre-commit` passes

**Health Checks:**
- [ ] Backend: https://my-backend-qa-pr-[PR_NUMBER].onrender.com/api/health
- [ ] Collect: https://my-collect-qa-pr-[PR_NUMBER].onrender.com/api/health
- [ ] Frontend: https://my-frontend-qa-pr-[PR_NUMBER].onrender.com
- [ ] Proxy: https://my-proxy-qa-pr-[PR_NUMBER].onrender.com/health