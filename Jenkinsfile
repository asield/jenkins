// Jenkinsfile OpenShift: Build (S2I) → Deploy → Service → Route TLS (edge + redirect)

pipeline {
  agent any

  tools {
    // Cliente "oc" registrado en Manage Jenkins → Global Tool Configuration
    oc 'oc'
  }

  environment {
    OPENSHIFT_PROJECT    = 'cicd-test'
    APP_NAME             = 'hello-go'
    GIT_REPO_URL         = 'https://github.com/asield/jenkins.git'
    OPENSHIFT_SERVER_URL = 'https://api.cluster-vwrjr.dynamic.redhatworkshops.io:6443'
  }

  stages {

    stage('⚙️ 1. Login a OpenShift') {
      steps {
        script {
          withCredentials([string(credentialsId: 'openshift-token-cicd', variable: 'OC_TOKEN')]) {
            // Evita interpolación Groovy del secreto: usa comillas simples triples
            sh label: 'oc login', script: '''
              oc login --token=$OC_TOKEN --server=$OPENSHIFT_SERVER_URL --insecure-skip-tls-verify=true
              oc project $OPENSHIFT_PROJECT
              echo "Login exitoso en el proyecto $OPENSHIFT_PROJECT"
            '''
          }
        }
      }
    }

    stage('📦 2. Crear BuildConfig (S2I)') {
      steps {
        script {
          def bcName = sh(
            script: "oc get bc/${APP_NAME} -n ${OPENSHIFT_PROJECT} -o name --ignore-not-found",
            returnStdout: true
          ).trim()

          if (!bcName) {
            echo "BuildConfig no encontrado. Creándolo (S2I Go toolset)..."
            // Opción S2I (builder soportado)
            sh """
              oc new-build --strategy=source \
                registry.redhat.io/ubi9/go-toolset:latest~${GIT_REPO_URL} \
                --name=${APP_NAME} -n ${OPENSHIFT_PROJECT}
            """
            // Alternativa (si usas Dockerfile en el repo), comenta lo de arriba y descomenta esto:
            // sh """
            //   oc new-build --strategy=docker ${GIT_REPO_URL} \
            //     --name=${APP_NAME} -n ${OPENSHIFT_PROJECT}
            // """
          } else {
            echo "BuildConfig '${bcName}' ya existe."
          }
        }
      }
    }

    stage('🚀 3. Iniciar Build') {
      steps {
        sh "oc start-build ${APP_NAME} -n ${OPENSHIFT_PROJECT} --follow || true"
        // En entornos con logs bloqueados, el --follow puede timeoutear; el build continúa en background.
      }
    }

    stage('🚢 4. Desplegar Aplicación') {
      steps {
        script {
          def deploymentName = sh(
            script: "oc get deployment/${APP_NAME} -n ${OPENSHIFT_PROJECT} -o name --ignore-not-found",
            returnStdout: true
          ).trim()

          def imageUrl = "image-registry.openshift-image-registry.svc:5000/${OPENSHIFT_PROJECT}/${APP_NAME}:latest"

          if (!deploymentName) {
            echo "Creando Deployment con imagen ${imageUrl}..."
            sh "oc create deployment ${APP_NAME} --image=${imageUrl} -n ${OPENSHIFT_PROJECT}"
            // Añade containerPort nombrado "http"
            sh """
              oc patch deploy ${APP_NAME} -n ${OPENSHIFT_PROJECT} -p '{
                "spec":{"template":{"spec":{"containers":[
                  {"name":"${APP_NAME}","ports":[{"containerPort":8080,"name":"http"}]}
                ]}}}
              }'
            """
          } else {
            echo "Deployment existente. Actualizando imagen..."
            sh "oc set image deployment/${APP_NAME} ${APP_NAME}=${imageUrl} -n ${OPENSHIFT_PROJECT}"
          }

          // Probes (opcional, recomendado)
          sh """
            oc set probe deploy/${APP_NAME} -n ${OPENSHIFT_PROJECT} \
              --readiness --get-url=http://:8080/ --initial-delay-seconds=3 --timeout-seconds=2 || true
          """
          sh """
            oc set probe deploy/${APP_NAME} -n ${OPENSHIFT_PROJECT} \
              --liveness  --get-url=http://:8080/ --initial-delay-seconds=10 --timeout-seconds=2 || true
          """

          sh "oc rollout status deployment/${APP_NAME} -n ${OPENSHIFT_PROJECT}"
        }
      }
    }

    stage('🌐 5. Exponer Servicio y Route (TLS edge + redirect)') {
      steps {
        script {
          // Crea el Service si no existe
          def svc = sh(
            script: "oc get svc/${APP_NAME} -n ${OPENSHIFT_PROJECT} -o name --ignore-not-found",
            returnStdout: true
          ).trim()
          if (!svc) {
            echo "Creando Service desde el deployment..."
            sh "oc expose deployment/${APP_NAME} --name=${APP_NAME} --port=8080 -n ${OPENSHIFT_PROJECT}"
          }

          // Normaliza puertos y selector del Service (JSON sin comentarios)
          sh """
            oc patch svc ${APP_NAME} -n ${OPENSHIFT_PROJECT} --type=merge -p '{
              "spec": {
                "ports": [
                  { "name": "http", "port": 8080, "targetPort": 8080, "protocol": "TCP" }
                ],
                "selector": { "app": "${APP_NAME}" }
              }
            }'
          """

          // Route TLS edge + Redirect apuntando al port name "http"
          def route = sh(
            script: "oc get route/${APP_NAME} -n ${OPENSHIFT_PROJECT} -o name --ignore-not-found",
            returnStdout: true
          ).trim()

          if (route) {
            sh "oc patch route ${APP_NAME} -n ${OPENSHIFT_PROJECT} -p '{\"spec\":{\"port\":{\"targetPort\":\"http\"}}}'"
            sh "oc patch route ${APP_NAME} -n ${OPENSHIFT_PROJECT} -p '{\"spec\":{\"tls\":{\"termination\":\"edge\",\"insecureEdgeTerminationPolicy\":\"Redirect\"}}}'"
          } else {
            sh """
              oc create route edge ${APP_NAME} \
                --service=${APP_NAME} \
                --port=http \
                --insecure-policy=Redirect \
                -n ${OPENSHIFT_PROJECT}
            """
          }

          def routeHost = sh(
            script: "oc get route ${APP_NAME} -n ${OPENSHIFT_PROJECT} -o jsonpath='{.spec.host}'",
            returnStdout: true
          ).trim()
          if (routeHost) {
            echo "URL segura: https://${routeHost}"
          }
        }
      }
    }
  }

  post {
    always {
      echo 'Pipeline finalizado. Cerrando sesión de OpenShift...'
      sh 'oc logout'
    }
    success {
      echo '¡Pipeline OK!'
    }
    failure {
      echo '¡El pipeline falló! Revisa los logs.'
    }
  }
}
