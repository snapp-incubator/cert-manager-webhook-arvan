[![Artifact HUB](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/hbahadorzadeh)](https://artifacthub.io/packages/helm/hbahadorzadeh/cert-manager-webhook-arvan)
# ACME webhook ArvanCloud

The ACME issuer type supports an optional 'webhook' solver, which can be used
to implement custom DNS01 challenge solving logic.

This is useful if you need to use cert-manager with a DNS provider that is not
officially supported in cert-manager core.

This plugin integrates with arvancloud api service to make DNS01 challenge possible for you.

Use helm to install this webhook:
```bash
helm repo add hbx https://hbahadorzadeh.github.io/helm-chart/
helm install -n cert-manager hbx/cert-manager-webhook-arvan
```

# Usage

You need to create a secret for your api key :

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: arvan-credentials
  namespace: cert-manager
stringData:
  apikey: "YOUR_API_KEY"
```

Sample ClusterIssuer:

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: test-issuer # Name of the issuer
  labels:
    app.kubernetes.io/name: test-issuer
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory # URL of the server 
    email: test@example.com #email of the user that will the notification about the cert 
    privateKeySecretRef:
      name: letsencrypt-account-key
    solvers:
    - dns01:
        webhook:
          groupName: hbahadorzadeh.github # name of the group you setted at the start of this course
          solverName: arvancloud
          config:
            ttl: 120
            authApiSecretRef: 
              name: "arvan-credentials"
              key":  "apikey"
            baseUrl: "https://napi.arvancloud.com"
```

Sample Certificate:

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: test-certificate # name of the certificate
  labels:
    app.kubernetes.io/name: test-certificate # name of the certificate
spec:
  dnsNames:
  - test.domain # name of the domain you want to validate the certificate
  issuerRef:
    name: test-issuer # name of the issuer you created before
    kind: ClusterIssuer
  secretName: test-certificate # name of the secret that will be created that will contain the certificate
```