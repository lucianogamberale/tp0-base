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

  Adicionalmente, se configuraron los volúmenes como read_only: true. Esto es una buena práctica de seguridad que asegura que las aplicaciones dentro de los contenedores puedan leer la configuración, pero no puedan modificar accidentalmente los archivos originales en la máquina host.

  De esta manera, la configuración queda desacoplada de la imagen. La imagen contiene la aplicación, pero los datos de configuración se "inyectan" en tiempo de ejecución desde el exterior.

- **Ejecución:**

  1.  Posicionarse en la rama: `git checkout ej2`

  2.  Levantar los servicios: `make docker-compose-up`

  3.  Modificar un valor en `config/server/config.ini` en la máquina _host_.

  4.  Reiniciar los servicios: `make docker-compose-down && make docker-compose-up`

  5.  Verificar en los logs (`make docker-compose-logs`) que el servidor ha tomado la nueva configuración sin necesidad de reconstruir la imagen.
