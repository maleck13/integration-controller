#!/bin/bash

# Replace branding with some new:
cat <<EOT | oc replace -f -
apiVersion: v1
data:
  config.json: |
    {
      "apiBase": "https://syndesis.192.168.64.50.nip.io",
      "apiEndpoint": "/api/v1",
      "title": "Syndesis Next Generation",
      "datamapper": {
        "baseMappingServiceUrl": "https://syndesis.192.168.64.50.nip.io/api/v1/atlas/",
        "baseJavaInspectionServiceUrl": "https://syndesis.192.168.64.50.nip.io/api/v1/atlas/java/",
        "baseXMLInspectionServiceUrl": "https://syndesis.192.168.64.50.nip.io/api/v1/atlas/xml/",
        "baseJSONInspectionServiceUrl": "https://syndesis.192.168.64.50.nip.io/api/v1/atlas/json/"
      },
      "features" : {
        "logging": false
      },
      "branding": {
        "logoWhiteBg": "",
        "logoDarkBg": "",
        "iconWhiteBg": "assets/images/glasses_logo_square.png",
        "iconDarkBg": "assets/images/glasses_logo_square.png",
        "appName": "Syndesis Next Generation",
        "favicon32": "/favicon-32x32.png",
        "favicon16": "/favicon-16x16.png",
        "touchIcon": "/apple-touch-icon.png",
        "productBuild": false
     }
    }
kind: ConfigMap
metadata:
  labels:
    app: syndesis
    component: syndesis-ui
  name: syndesis-ui-config
  namespace: myproject
EOT
