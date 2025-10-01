# Hello Go on OpenShift – CI/CD con Jenkins (S2I → Deploy → Service → Route TLS)

Este proyecto implementa un pipeline de Jenkins que construye y despliega una aplicación **Go** en **OpenShift** usando **Source-to-Image (S2I)**, y la expone mediante **Service** y **Route** con **TLS (edge)** y **redirección HTTP→HTTPS**.

## Diagrama de alto nivel

1. **Login** a OpenShift con token de servicio.  
2. **Build S2I** con `ubi9/go-toolset` apuntando al repo Git.  
3. **Deploy** de la imagen `:latest` al proyecto destino.  
4. **Service** TCP/8080 (puerto nombrado `http`).  
5. **Route** TLS edge con `insecureEdgeTerminationPolicy=Redirect`.

## Requisitos

- Jenkins con:
  - Plugin **Pipeline**.
  - Herramienta **oc** registrada en *Manage Jenkins → Global Tool Configuration* (nombre: `oc`).
  - Credencial **Secret text** con ID `openshift-token-cicd` (token con permisos en el proyecto).
- Proyecto de OpenShift existente (por defecto: `cicd-test`).
- Acceso a `registry.redhat.io` (si usas el builder `ubi9/go-toolset`). Si no, sustituye por otro builder soportado o usa estrategia Docker.

---

## Configuración del ServiceAccount y Token (Opción A – recomendada)

En OpenShift, lo mejor es asociar un **ServiceAccount (SA)** a un rol ya existente (`edit` o `view`) en el *namespace*, en lugar de crear roles personalizados.

### 1. Crear el ServiceAccount
```bash
oc create sa jenkins-cicd -n cicd-test
```

### 2. Asignar permisos (ejemplo: rol `edit`)
```bash
# Permite a Jenkins crear/actualizar la mayoría de recursos en el proyecto
oc policy add-role-to-user edit -z jenkins-cicd -n cicd-test
```

> Si solo necesitas lectura, usa `view` en lugar de `edit`.

### 3. Generar un token para Jenkins
OpenShift 4.11+ permite crear tokens con vencimiento:
```bash
oc create token jenkins-cicd -n cicd-test --duration=8760h
```

Ese token se debe registrar en Jenkins como **Secret Text Credential** con el ID `openshift-token-cicd`.

---

## Variables del Pipeline

Configura estas variables en el `Jenkinsfile` o como **parámetros** del job:

- `OPENSHIFT_PROJECT`: proyecto/namespace destino (por defecto `cicd-test`).
- `APP_NAME`: nombre de la app y de los recursos (`hello-go`).
- `GIT_REPO_URL`: repositorio Git con el código fuente (este repo u otro).
- `OPENSHIFT_SERVER_URL`: URL de la API del cluster.

## Estructura del Pipeline

- **1. Login a OpenShift**  
  Autentica con `oc login` usando el token `openshift-token-cicd` y selecciona el proyecto.

- **2. Crear BuildConfig (S2I)**  
  Crea `BuildConfig` S2I con `ubi9/go-toolset` si no existe.  
  > Alternativa: estrategia **Docker** si el repo contiene `Dockerfile` (ver bloque comentado en el Jenkinsfile).

- **3. Iniciar Build**  
  Ejecuta `oc start-build ${APP_NAME} --follow`. El build continúa aunque `--follow` tenga timeout.

- **4. Desplegar Aplicación**  
  Crea o actualiza un **Deployment** con la imagen:  
  `image-registry.openshift-image-registry.svc:5000/${OPENSHIFT_PROJECT}/${APP_NAME}:latest`  
  Añade puerto 8080 nombrado **http** y configura **readiness/liveness** probes.

- **5. Exponer Servicio y Route (TLS)**  
  Crea/ajusta el **Service** (puerto 8080, nombre `http`) y la **Route** con `termination=edge` + `Redirect`.  
  Muestra la URL final `https://<host>`.

## Ejecución

1. Crea un **Multibranch Pipeline** o **Pipeline** en Jenkins que apunte a este repo (o copia el `Jenkinsfile` en tu job).
2. Verifica:
   - Herramienta `oc` disponible.
   - Credencial `openshift-token-cicd`.
   - Variables del entorno en el Jenkinsfile (proyecto, API, etc.).
3. Ejecuta el pipeline.  
   Al finalizar, verás en consola la **URL segura** de la aplicación.

## Troubleshooting

- **Fallo al loguear (`oc login`)**:  
  - Revisa la URL de la API (`OPENSHIFT_SERVER_URL`) y el token.  
  - Si tu cluster usa un certificado válido, puedes quitar `--insecure-skip-tls-verify=true`.

- **Error en S2I / acceso al builder**:  
  - Asegura el pull secret para `registry.redhat.io` en el proyecto o usa un builder diferente (por ejemplo, `ubi9` + toolchain) o la estrategia Docker.

- **Route no redirige a HTTPS**:  
  - Verifica `spec.tls.termination=edge` y `insecureEdgeTerminationPolicy=Redirect`.  
  - Asegúrate de que el **Service** tiene el puerto **nombrado** `http`.

- **Probes fallando**:  
  - Ajusta `initial-delay-seconds` y `timeout-seconds`.  
  - Verifica que la app responda en `/` y puerto 8080.

## Personalización

- **Parámetros de Job**: Puedes migrar las variables a parámetros de Jenkins para no editar el Jenkinsfile.
- **Dockerfile**: Si prefieres construir con Docker/Buildah, usa la estrategia Docker (bloque comentado) y asegúrate de que el `Dockerfile` expone el puerto 8080.
- **TLS Passthrough/Reencryption**: Cambia `oc create route edge` por `passthrough` o `reencrypt` según tus necesidades y certificados.

## Limpieza

Para eliminar los recursos creados por la app:
```bash
oc delete route,svc,deploy,bc,builds -l app=hello-go -n <proyecto>
# o puntualmente:
oc delete route/hello-go svc/hello-go deploy/hello-go bc/hello-go -n <proyecto>
```

---

## Licencia

Este proyecto se distribuye bajo la licencia MIT:

```
MIT License

Copyright (c) 2025 

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
```
