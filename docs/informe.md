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

