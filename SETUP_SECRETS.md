# GitHub Actions Secrets Setup

To enable the release workflow, add these secrets to your GitHub repository:

## Steps
1. Go to: https://github.com/zhf883680/clash-subscription-manager/settings/secrets/actions
2. Click "New repository secret"
3. Add the following secrets:

### DOCKER_USERNAME
- Name: `DOCKER_USERNAME`
- Value: `zhf883680`

### DOCKER_PASSWORD
- Value: Your Docker Hub Access Token

## Getting Docker Hub Access Token
1. Go to https://hub.docker.com/settings/security
2. Click "New Access Token"
3. Give it a description (e.g., "GitHub Actions - clash-subscription-manager")
4. Select "Read & Write" access
5. Copy the token
6. Paste it as the DOCKER_PASSWORD secret value

**Security Note:** Never use your Docker Hub account password. Use access tokens which can be revoked anytime.