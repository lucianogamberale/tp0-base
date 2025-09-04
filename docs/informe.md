# Informe de Resolución - TP0

Este documento detalla la estrategia de resolución para cada ejercicio del Trabajo Práctico N°0, junto con las decisiones de diseño clave sobre el protocolo de comunicación y los mecanismos de concurrencia implementados.

---

## Ejecución y Detalles de la Solución

A continuación se detalla la estrategia de resolución para cada ejercicio, junto con las instrucciones para ejecutar y verificar el funcionamiento de cada etapa.

Cada ejercicio se encuentra en su propia rama de Git. Para probar una solución específica, primero debe posicionarse en la rama correspondiente, por ejemplo: `git checkout ej1`.

---

### Ejercicio 1: Generar Docker Compose configurable

- **Objetivo:**

  Crear un script `generar-compose.sh` que genere un archivo `docker-compose.yaml` con una cantidad configurable de clientes.

- **Implementación:**

  Se desarrolló un script en `bash` que permite recibir dos parámetros de entrada: el nombre del archivo de salida y la cantidad de clientes.

  Chequeo de parámetros mandatorios: se verifican los parámetros mandatorios, y de no cumplirse se brinda una explicación de uso y se retorna un error. Además, se verifica que el número de clientes sea válido (que sea un número mayor o igual a cero).

  Modularidad y claridad: se separó el script en funciones pequeñas y con un propósito único (`add-server-service`, `add-client-service`, etc.) que intentan mantener la estructura de un `compose.yaml`, con el objetivo de que sea escalable y fácil de modificar.

- **Ejecución:**

  1.  Posicionarse en la rama: `git checkout ej1`

  2.  Ejecutar el script para generar un archivo, por ejemplo, `docker-compose-dev.yaml` con 5 clientes:

      ```bash
      ./generar-compose.sh docker-compose-dev.yaml 5
      ```

  3.  Levantar los servicios utilizando el archivo generado:

      ```bash
      make docker-compose-up

      ```

### Ejercicio 2: Configuración Externa

- **Objetivo:**

  Modificar el sistema para que los cambios en los archivos de configuración (`config.ini` para el servidor y `config.yaml` para el cliente) no requieran reconstruir las imágenes de Docker.

- **Implementación:**

  La solución se implementó actualizando el script `generar-compose.sh` para que incluya volúmenes de tipo bind mount en la definición de los servicios. Un bind mount crea un enlace directo entre un archivo o directorio del sistema anfitrión (host) y el sistema de archivos del contenedor.

  Se añadieron las siguientes configuraciones de volúmenes:

  - **Servidor**: Se mapea el archivo local `./server/config.ini` al archivo `/config.ini` dentro del contenedor del servidor.
  - **Cliente**: Se mapea `./client/config.yaml` al archivo `/config.yaml` dentro de cada contenedor de cliente.

  Adicionalmente, se configuraron los volúmenes como `read_only: true`. Esto es una buena práctica de seguridad que asegura que las aplicaciones dentro de los contenedores puedan leer la configuración, pero no puedan modificar accidentalmente los archivos originales en la máquina host.

  De esta manera, la configuración queda desacoplada de la imagen. La imagen contiene la aplicación, pero los datos de configuración se "inyectan" en tiempo de ejecución desde el exterior.

- **Ejecución:**

  1.  Posicionarse en la rama: `git checkout ej2`

  2.  Levantar los servicios: `make docker-compose-up`

  3.  Modificar un valor en `config/server/config.ini` en la máquina _host_.

  4.  Verificar dentro de del container correspondiente que se modificó el archivo. Para esto nos adentramos en el container `docker exec -it server sh` y realizar un `cat config.ini`.

### Ejercicio 3: Script de Validación con Netcat

- **Objetivo:** Crear un script `validar-echo-server.sh` para verificar que el servidor funciona correctamente como un _echo server_. La validación debe usar `netcat` desde dentro de la red de Docker, sin exponer puertos al _host_.

- **Implementación:**
  La solución adopta un enfoque de **"tester en un contenedor"**. En lugar de ejecutar `netcat` desde el _host_ o desde uno de los contenedores de cliente, se crea un entorno de prueba dedicado y efímero.

  La implementación se divide en tres partes:

  1.  **`Dockerfile` del Netcat Echo Server Tester:** Se creó un `Dockerfile` minimalista que utiliza `alpine` como imagen base. Alpine es una distribución de Linux extremadamente ligera que ya incluye `netcat` por defecto. Este Dockerfile simplemente copia y da permisos de ejecución al script que contiene la lógica de la prueba.

  2.  **Script de Lógica (`netcat-echo-sv-test.sh`):** Este script es el que se ejecuta _dentro_ del contenedor de prueba. Su lógica es simple:

      - Define un mensaje de prueba.
      - Usa `netcat` (`nc`) para enviar ese mensaje al servidor, direccionándolo por su nombre de servicio en la red Docker (`server:12345`).
      - Toma la respuesta del servidor.
      - Compara la respuesta con el mensaje original y, según el resultado, imprime el log de `success` o `fail` requerido.

  3.  **Script Orquestador (`validar-echo-server.sh`):** Este es el script principal que se ejecuta desde el _host_. Su función es orquestar todo el proceso de prueba:
      - Primero, construye la imagen del tester usando el comando `docker build` con un tag parametrizable.
      - Luego, ejecuta un contenedor temporal (`docker run --rm`) a partir de esa imagen. Este contendor se conecta a la red del proyecto (`--network=tp0_testing_net`), lo que le permite resolver el nombre `server` y comunicarse con él.
      - Finalmente, se le indica al contenedor que ejecute el script de lógica (`sh -c "./$script_name"`).

- **Ejecución:**
  1.  Posicionarse en la rama: `git checkout ej3`
  2.  Asegurarse de que el entorno principal esté funcionando:
      ```bash
      make docker-compose-up
      ```
  3.  En otra terminal, dar permisos de ejecución y correr el script de validación:
      ```bash
      chmod +x validar-echo-server.sh
      ./validar-echo-server.sh
      ```
  4.  El script se encargará de construir la imagen de prueba, ejecutar el test y mostrar el resultado `action: test_echo_server | result: success` en la consola.
