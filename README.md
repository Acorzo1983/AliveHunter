# AliveHunter

AliveHunter es una herramienta escrita en Go para comprobar si las URLs están vivas (responden con un estado HTTP 200 OK). Está diseñado para trabajar con listas de subdominios y admite el uso de proxies para distribuir las solicitudes.

## Características

- Verificación de URLs en bloques para mejorar la eficiencia.
- Admite múltiples reintentos y tiempo de espera configurable.
- Admite proxies para distribuir la carga de las solicitudes.
- Registro de errores detallado en `error_log.txt`.
- Barra de progreso para visualizar el estado del proceso.

## Requisitos

- Go 1.16 o superior
- Las siguientes bibliotecas de Go:
  - `github.com/fatih/color`
  - `github.com/schollz/progressbar/v3`

Para instalar las dependencias, ejecuta:

```sh
go get github.com/fatih/color
go get github.com/schollz/progressbar/v3
```

## Instalación

Para instalar `AliveHunter`, descarga el archivo `installer.sh`, hazlo ejecutable y ejecútalo:

```sh
chmod +x installer.sh
./installer.sh
```

## Uso

### Ejemplos de Uso

1. **Uso básico**:

```sh
go run AliveHunter.go -l subdomainlist.txt
```

2. **Guardar los resultados en un archivo específico**:

```sh
go run AliveHunter.go -l subdomainlist.txt -o alive_subdomains.txt
```

3. **Uso con una lista de proxies**:

```sh
go run AliveHunter.go -l subdomainlist.txt -p proxylist.txt
```

4. **Configurar el número de reintentos y el tiempo de espera**:

```sh
go run AliveHunter.go -l subdomainlist.txt -r 5 -t 15
```

5. **Dividir las URLs en un número máximo de bloques**:

```sh
go run AliveHunter.go -l subdomainlist.txt -b 100
```

6. **Comprobar solo URLs HTTPS**:

```sh
go run AliveHunter.go -l subdomainlist.txt --https
```

### Ejemplo Completo con Subfinder

Para usar `AliveHunter` junto con `subfinder`:

1. Genera el archivo de subdominios con `subfinder`:

```sh
subfinder -d example.com --silent -o subdomainlist.txt
```

2. Ejecuta `AliveHunter` con el archivo generado:

```sh
go run AliveHunter.go -l subdomainlist.txt -o alive_subdomains.txt
```

### Uso de Proxychains

Para usar `proxychains` para multiproxying:

```sh
proxychains go run AliveHunter.go -l subdomainlist.txt
```

### Ayuda

Para mostrar el mensaje de ayuda:

```sh
go run AliveHunter.go -h
```

## Opciones

```
-l string
      File containing URLs to check (required)
-o string
      Output file to save the results (optional, default is <input_file>_alive.txt)
-p string
      File containing proxy list (optional)
-r int
      Number of retries for failed requests (default 5)
-t int
      Timeout for HTTP requests in seconds (default 15)
-b int
      Maximum number of blocks to divide (default 1000)
--https
      Check only HTTPS URLs
-h    Show help message
```

## Hecho con ❤️ por Albert.C
