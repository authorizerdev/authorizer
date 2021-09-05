<p align="center">
  <a href="https://authorizer.dev">
    <img alt="Logo" src="https://github.com/authorizerdev/authorizer/blob/main/assets/logo.png" width="60" />
  </a>
</p>
<h1 align="center">
  Authorizer
</h1>


**Authorizer** is an open-source authentication and authorization solution for your applications. Bring your database and have complete control over the user information. You can self-host authorizer instances and connect to any SQL database.

## Table of contents
- [Introduction](#introduction)
- [Getting Started](#getting-started)
- [Contributing](https://github.com/authorizerdev/authorizer/blob/main/.github/CONTRIBUTING.md)
- [Docs](http://docs.authorizer.dev/)
- [Join Community](https://discord.gg/2fXUQN3E)

# Introduction

<img src="https://github.com/authorizerdev/authorizer/blob/main/assets/authorizer-architecture.png" style="height:20em"/>

#### We offer the following functionality

- ✅ Sign-in / Sign-up with email ID and password
- ✅ Secure session management
- ✅ Email verification
- ✅ APIs to update profile securely
- ✅ Forgot password flow using email
- ✅ Social logins (Google, Github, more coming soon)

## Project Status

⚠️ **Authorizer is still an early beta! missing features and bugs are to be expected!** If you can stomach it, then bring authentication and authorization to your site today!

## Roadmap

- Password-less login with email and magic link
- Role-based access management system
- Support more JWT encryption algorithms (Currently supporting HS256)
- 2 Factor authentication
- Back office (Admin dashboard to manage user)
- Support more database
- VueJS SDK
- Svelte SDK
- React Native SDK
- Flutter SDK
- Android Native SDK
- iOS native SDK
- Golang SDK
- Python SDK
- PHP SDK
- WordPress plugin
- Kubernetes Helm Chart
- [Local Stack](https://github.com/localstack/localstack)
- AMI
- Digital Ocean Droplet
- Azure
- Render
- Edge Deployment using Fly.io
- Password-less login with mobile number and OTP SMS

# Getting Started

## Trying out Authorizer

This guide helps you practice using Authorizer to evaluate it before you use it in a production environment. It includes instructions for installing the Authorizer server in standalone mode.

## Installing a simple instance of Authorizer

Deploy Authorizer using [heroku](https://github.com/authorizerdev/authorizer-heroku) and quickly play with it in 30seconds
<br/><br/>
[![Deploy to Heroku](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy?template=https://github.com/authorizerdev/authorizer-heroku)

### Things to consider

- For social logins, you will need respective social platform key and secret
- For having verified users, you will need an SMTP server with an email address and password using which system can send emails. The system will send a verification link to an email address. Once an email is verified then, only able to access it.
  > Note: One can always disable the email verification to allow open sign up, which is not recommended for production as anyone can use anyone's email address 😅
- For persisting user sessions, you will need Redis URL. If you do not configure a Redis server, sessions will be persisted until the instance is up or not restarted. For better response time on authorization requests/middleware, we recommend deploying Redis on the same infra/network as your authorizer server.

## Integrating into your website

This example demonstrates how you can use [`@authorizerdev/authorizer-js`](/authorizer-js/getting-started) CDN version and have login ready for your site in few seconds. You can also use the ES module version of [`@authorizerdev/authorizer-js`](/authorizer-js/getting-started) or framework-specific versions like [`@authorizerdev/authorizer-react`](/authorizer-react/getting-started)

### Copy the following code in `html` file

> **Note:** Change AUTHORIZER_URL in the below code with your authorizer URL. Also, you can change the logout button component

```html
<script src="https://unpkg.com/@authorizerdev/authorizer-js/lib/authorizer.min.js"></script>

<script type="text/javascript">
  const authorizerRef = new authorizerdev.Authorizer({
    authorizerURL: `AUTHORIZER_URL`,
    redirectURL: window.location.origin,
  });

  // use the button selector as per your application
  const logoutBtn = document.getElementById("logout");
  logoutBtn.addEventListener("click", async function () {
    await authorizerRef.logout();
    window.location.href = "/";
  });

  async function onLoad() {
    const res = await authorizerRef.fingertipLogin();
    if (res && res.user) {
      // you can use user information here, eg:
      /**
      const userSection = document.getElementById('user');
      const logoutSection = document.getElementById('logout-section');
      logoutSection.classList.toggle('hide');
      userSection.innerHTML = `Welcome, ${res.user.email}`;
      */
    }
  }
  onLoad();
</script>
```
