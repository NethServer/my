# 🔐 Nethesis Operation Center (aka my.nethesis.it)

A web application that provides centralized authentication using [Logto](https://logto.io/) as an Identity Provider (IdP), with:

- 🌐 A **Vue** frontend  
- 🛠️ A **Go** backend  
- 🧩 External app login support (WordPress, Moodle, Freshdesk...) via **OIDC/SAML**

---

## 📦 Structure

```
/
├── frontend/
│   └── (Vue 3 app using Logto JS SDK)
├── backend/
│   └── (Go API protected by Logto access tokens)
└── README.md
```

---

## 📸 Architecture overview

This project implements a complete authentication flow:

1. Users register/login in the Vue frontend  
2. Logto handles sessions and issues tokens  
3. The Go backend validates tokens before serving protected API requests  
4. Other platforms (WordPress, etc.) use Logto as external login via OpenID Connect  

![image](https://github.com/user-attachments/assets/54450f7f-8313-455c-a320-21e3b0f1bf32)


---

## 🚀 Features

- ✅ IdP login & registration  
- ✅ Secure token validation in backend  
- ✅ SSO login from external apps  

---

## 🧱 Tech Stack

- [Logto](https://logto.io/)
- [Vue 3 + Vite](https://vitejs.dev/)
- [Go 1.20+](https://golang.org/)
- [Logto JS SDK](https://docs.logto.io/recipes/vue/)
- [Logto Go SDK](https://docs.logto.io/recipes/go/)

---

## 🛠️ Setup (WIP)

Coming soon:

- [ ] Local dev instructions  
- [ ] Logto environment setup  
- [ ] Environment variable configuration  

---

## 📦 Build & CI (WIP)


---

## 🙏 Acknowledgements

- Based on [Logto](https://github.com/logto-io/logto)  
- Inspired by modern authentication best practices  
