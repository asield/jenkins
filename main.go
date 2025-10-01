// main.go
package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
)

func main() {
    // OpenShift S2I para Go espera que la app corra en el puerto 8080.
    port := "8080"
    log.Printf("Iniciando servidor en el puerto %s...\n", port)

    // Define el manejador para la ruta principal "/"
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        // Obtenemos el nombre del Host (del Pod) para mostrarlo
        hostname, _ := os.Hostname()
        fmt.Fprintf(w, "Hello World from Go!\nServidor corriendo en: %s\n", hostname)
    })

    // Inicia el servidor HTTP
    if err := http.ListenAndServe(":"+port, nil); err != nil {
        log.Fatal(err)
    }
}
