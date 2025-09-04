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

### Ejercicio 4: Graceful Shutdown

- **Objetivo:** Modificar el cliente y el servidor para que ambos manejen la señal `SIGTERM` y terminen de forma ordenada. Un cierre _graceful_ implica liberar todos los recursos adquiridos, como sockets y archivos, antes de que el proceso principal finalice.

- **Implementación (Cliente - Go):**
  La gestión de señales en el cliente se implementó utilizando un channel que sirve de buzón para recibir señales de `SIGTERM`.

  1.  **Canal de Señales:** Se crea un canal (`make(chan os.Signal, 1)`) que actúa como un buzón para recibir notificaciones de señales enviadas por el sistema operativo.

  2.  **Registro de Señal:** Usando `signal.Notify(channel, syscall.SIGTERM)`, se le indica al runtime de Go que debe enviar una notificación al canal cada vez que el proceso reciba la señal `SIGTERM`.

  3.  **Manejo con `select`:** El bucle principal del cliente, que se encarga de enviar mensajes periódicamente, se modificó para usar una sentencia `select`. En cada iteración, el `select` permite al programa esperar por dos eventos de forma no bloqueante:

      - **Caso 1 (Señal Recibida):** Si llega un mensaje al canal de señales, significa que se recibió `SIGTERM`. En este caso, se ejecuta la función de limpieza `sigtermSignalHandler()` y se termina el bucle principal, finalizando el programa de forma controlada.
      - **Caso 2 (`default`):** Si no hay ninguna señal pendiente, se ejecuta el bloque `default`, que contiene la lógica normal de crear un socket, enviar un mensaje y esperar el período de tiempo configurado.

  4.  **Lógica de Limpieza (`sigtermSignalHandler`):** Esta función se encarga de la limpieza de recursos. Su responsabilidad es verificar si existe una conexión de red activa (`client.conn != nil`) y, de ser así, cerrarla explícitamente con `client.conn.Close()`. Además, se utiliza el comando `defer` para asegurarnos de que siempre se cierre el canal de señales y también la conexión con el cliente.

  Este diseño asegura que si se ejecuta un `docker compose down` (que envía `SIGTERM` a los contenedores), el cliente lo interceptará, cerrará su socket de red y terminará limpiamente.

- **Implementación (Servidor - Python):**
  El manejo de la señal `SIGTERM` en el servidor se realizó utilizando el módulo nativo `signal` de Python.

  1.  **Registro del Manejador:** En el constructor de la clase `Server`, se registra un manejador (`__sigterm_signal_handler`) para la señal `SIGTERM` mediante la llamada `signal.signal(signal.SIGTERM, ...)`.

  2.  **Flag de Control:** La clase utiliza una variable booleana interna (`self._server_running`) que actúa como un _flag_ para controlar la ejecución del bucle principal en el método `run()`.

  3.  **Lógica del Manejador (`__sigterm_signal_handler`):** Cuando el sistema operativo envía la señal `SIGTERM`, se invoca este método, que realiza dos acciones clave en secuencia:
      - Primero, cambia el estado del _flag_ `_server_running` a `False`. Esto asegura que, una vez que el bucle principal termine su iteración actual, no comience una nueva.
      - Segundo, y más importante, cierra el socket principal del servidor (`self._server_socket.close()`). Esta acción es fundamental porque el bucle principal está bloqueado en la llamada `self._server_socket.accept()`. Al cerrar el socket desde el manejador de la señal, la llamada `accept()` falla inmediatamente, levantando una `OSError`. Esto **desbloquea el hilo principal** y le permite re-evaluar la condición del bucle `while`, que ahora es `False`, llevando a una terminación controlada del servidor.

  Esta combinación de un _flag_ de estado y el cierre forzado del socket para interrumpir una llamada bloqueante es un patrón robusto y efectivo para lograr un apagado _graceful_ en servidores de red.

- **Ejecución:**

  1.  Posicionarse en la rama: `git checkout ej4`

  2.  Levantar los servicios:

      ```bash
      make docker-compose-up
      ```

  3.  Mientras los servicios corren, detenerlos con el comando de make:

      ```bash
      make docker-compose-down
      ```

  4.  Revisar los logs con `make docker-compose-logs`. Se deberán observar los mensajes de log definidos en los manejadores de señales tanto del cliente (`action: sigterm_signal_handler`) como del servidor, indicando que los recursos se cerraron correctamente antes de que los contenedores se detuvieran. Output de ejemplo:

      ```bash
      client1  | 2025-09-04 17:42:55 INFO     action: config | result: success | client_id: 1 | server_address: server:12345 | loop_amount: 5 | loop_period: 5s | log_level: DEBUG
      client1  | 2025-09-04 17:42:55 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°1
      server   | 2025-09-04 17:42:55 DEBUG    action: config | result: success | port: 12345 | listen_backlog: 5 | logging_level: DEBUG
      server   | 2025-09-04 17:42:55 INFO     action: server_startup | result: success
      server   | 2025-09-04 17:42:55 INFO     action: accept_connections | result: in_progress
      server   | 2025-09-04 17:42:55 INFO     action: accept_connections | result: success | ip: 172.25.125.3
      server   | 2025-09-04 17:42:55 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°1
      server   | 2025-09-04 17:42:55 INFO     action: send_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°1
      server   | 2025-09-04 17:42:55 DEBUG    action: client_connection_close | result: success
      server   | 2025-09-04 17:42:55 INFO     action: accept_connections | result: in_progress
      client1  | 2025-09-04 17:43:00 DEBUG     action: client_connection_close | result: success | client_id: 1
      server   | 2025-09-04 17:43:00 INFO     action: accept_connections | result: success | ip: 172.25.125.3
      server   | 2025-09-04 17:43:00 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°2
      server   | 2025-09-04 17:43:00 INFO     action: send_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°2
      client1  | 2025-09-04 17:43:00 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°2
      server   | 2025-09-04 17:43:00 DEBUG    action: client_connection_close | result: success
      server   | 2025-09-04 17:43:00 INFO     action: accept_connections | result: in_progress
      server   | 2025-09-04 17:43:05 INFO     action: accept_connections | result: success | ip: 172.25.125.3
      server   | 2025-09-04 17:43:05 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°3
      client1  | 2025-09-04 17:43:05 DEBUG     action: client_connection_close | result: success | client_id: 1
      client1  | 2025-09-04 17:43:05 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°3
      server   | 2025-09-04 17:43:05 INFO     action: send_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°3
      server   | 2025-09-04 17:43:05 DEBUG    action: client_connection_close | result: success
      server   | 2025-09-04 17:43:05 INFO     action: accept_connections | result: in_progress
      client1  | 2025-09-04 17:43:10 DEBUG     action: client_connection_close | result: success | client_id: 1
      server   | 2025-09-04 17:43:10 INFO     action: accept_connections | result: success | ip: 172.25.125.3
      server   | 2025-09-04 17:43:10 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°4
      server   | 2025-09-04 17:43:10 INFO     action: send_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°4
      client1  | 2025-09-04 17:43:10 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°4
      server   | 2025-09-04 17:43:10 DEBUG    action: client_connection_close | result: success
      server   | 2025-09-04 17:43:10 INFO     action: accept_connections | result: in_progress
      client1  | 2025-09-04 17:43:15 DEBUG     action: client_connection_close | result: success | client_id: 1
      server   | 2025-09-04 17:43:15 INFO     action: accept_connections | result: success | ip: 172.25.125.3
      server   | 2025-09-04 17:43:15 INFO     action: receive_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°5
      client1  | 2025-09-04 17:43:15 INFO     action: receive_message | result: success | client_id: 1 | msg: [CLIENT 1] Message N°5
      server   | 2025-09-04 17:43:15 INFO     action: send_message | result: success | ip: 172.25.125.3 | msg: [CLIENT 1] Message N°5
      server   | 2025-09-04 17:43:15 DEBUG    action: client_connection_close | result: success
      server   | 2025-09-04 17:43:15 INFO     action: accept_connections | result: in_progress
      client1  | 2025-09-04 17:43:20 DEBUG     action: client_connection_close | result: success | client_id: 1
      client1  | 2025-09-04 17:43:20 INFO     action: loop_finished | result: success | client_id: 1
      client1  | 2025-09-04 17:43:20 DEBUG     action: signal_channel_close | result: success | client_id: 1
      client1  | 2025-09-04 17:43:20 INFO     action: exit | result: success | client_id: 1
      client1 exited with code 0
      server   | 2025-09-04 17:43:41 INFO     action: sigterm_signal_handler | result: in_progress
      server   | 2025-09-04 17:43:41 DEBUG    action: sigterm_server_socket_close | result: success
      server   | 2025-09-04 17:43:41 INFO     action: sigterm_signal_handler | result: success
      server   | 2025-09-04 17:43:41 ERROR    action: accept_connections | result: fail | error: [Errno 9] Bad file descriptor
      server   | 2025-09-04 17:43:41 DEBUG    action: server_socker_close | result: success
      server   | 2025-09-04 17:43:41 INFO     action: server_shutdown | result: success
      server   | 2025-09-04 17:43:41 INFO     action: exit | result: success
      ```

### Ejercicio 5: Lotería Nacional - Apuesta Simple

- **Objetivo:** Adaptar el sistema al caso de uso de la "Lotería Nacional". Los clientes (agencias) deben enviar los datos de una apuesta individual (leída desde variables de entorno) al servidor, que se encargará de almacenarla. La comunicación debe seguir un protocolo definido.

- **Implementación (Cliente - Go):**
  La lógica del cliente fue refactorizada para enviar una única apuesta estructurada y verificar su correcta recepción por parte del servidor.

  1.  **Modelo de Dominio (`Bet`):** Se crea el struct `Bet` que modela los datos de una apuesta (agencia, nombre, DNI, etc.). La carga de esta estructura se realizar a partir de variables de entorno que fueron configuradas en el `docker-compose-dev.yaml`.

  2.  **Protocolo de Comunicación:** El cambio más significativo es la implementación de un protocolo de comunicación formal, encapsulado en el package `communicationProtocol.go`. Este protocolo define la estructura de todos los mensajes intercambiados con el servidor. El protocolo de comunicación se explicará con mayor lujo de detalle mas adealante.

  3.  **Serialización de Datos:** Antes de enviar una apuesta, los datos de la struct `Bet` se serializan a un formato de texto específico definido por el protocolo. La función `EncodeBetMessage` se encarga de transformar la apuesta en un string con el formato `BET[{"campo":"valor",...}]`.

  4.  **Flujo de Envío y Confirmación (ACK):** El flujo de comunicación para enviar una apuesta es ahora más robusto:
      - El cliente envía el mensaje serializado de la apuesta.
      - A continuación, **espera activamente una respuesta** del servidor.
      - Se implementó una verificación de **Acknowledgement (ACK)**. El cliente espera recibir un mensaje específico del servidor (en este caso, `ACK[1]`) que confirma que la apuesta fue recibida y procesada correctamente.
      - Si la respuesta no es el ACK esperado, el cliente registra un error. De lo contrario, imprime el log de éxito `action: apuesta_enviada | result: success ...` requerido por la consigna.

  Este mecanismo de ACK es fundamental, ya que le da al cliente la certeza de que su operación fue completada con éxito en el servidor, en lugar de solo enviarla "a ciegas".

- **Implementación (Servidor - Python):**
  El servidor fue modificado para entender el nuevo protocolo, procesar las apuestas recibidas y confirmar su recepción.

  1.  **Adherencia al Protocolo:** El servidor ahora utiliza un módulo `communication_protocol.py` que contiene la lógica para decodificar los mensajes entrantes.

  2.  **Deserialización de Datos:** Al recibir datos de un cliente, el servidor realiza el proceso inverso al del cliente:

      - Verifica que el tipo de mensaje sea `BET`.
      - Extrae el `PAYLOAD` del mensaje.
      - Parsea la cadena de texto `{"campo":"valor",...}` para reconstruir un objeto `Bet` en memoria con todos los datos de la apuesta.

  3.  **Lógica de Negocio y Persistencia:** Una vez que la apuesta es deserializada y validada, el servidor invoca la función provista `utils.store_bets()` para almacenar la apuesta de forma persistente. A continuación, emite el log requerido: `action: apuesta_almacenada | result: success ...`.

  4.  **Envío de Confirmación (ACK):** Para completar el flujo, después de almacenar la apuesta, el servidor construye y envía un mensaje `ACK[1]` al cliente. Este paso es crucial para notificarle al cliente que la operación fue exitosa.

  5.  **Manejo de I/O Robusto:** Se mejoró el manejo de la comunicación para evitar fenómenos de lecturas/escrituras cortas (_short-reads/writes_). La recepción de mensajes se realiza en un bucle que acumula _chunks_ hasta recibir un delimitador, y el envío se realiza con `sendall()`, que garantiza el envío de todos los bytes.

- **Ejecución:**
  1.  Posicionarse en la rama: `git checkout ej5`.
  2.  Levantar los servicios: `make docker-compose-up`.
  3.  Observar los logs (`make docker-compose-logs`). Se verá la secuencia de logs `apuesta_enviada` de cada cliente, seguida de su correspondiente `apuesta_almacenada` en el servidor. Una vez que el servidor procese las 5 apuestas, finalizará su ejecución.

---

## Aspectos de Diseño

### Protocolo de Comunicación

Para este trabajo práctico se diseñó e implementó un protocolo de comunicación basado en el intercambio de mensajes con una estructura definida.

- **Formato General del Mensaje:**
  Todos los mensajes siguen la estructura `TIPO[PAYLOAD]`, donde:

  - **`TIPO`**: Un string de 3 caracteres que identifica la naturaleza del mensaje (ej: `BET`, `ACK`).
  - **`PAYLOAD`**: El contenido del mensaje, delimitado por corchetes `[]`.

- **Tipos de Mensajes (Ejercicio 5):**

  - **`BET`**: Mensaje enviado por el cliente para registrar una o más apuestas.
    - **Payload:** Una o más apuestas serializadas. Para una única apuesta, el formato es `{"agency":"<valor>","first_name":"<valor>",...}`.
  - **`ACK`**: Mensaje enviado por el servidor para confirmar la recepción y procesamiento exitoso de un mensaje previo.
    - **Payload:** Generalmente un número que indica la cantidad de items procesados. Para este ejercicio, es `1`.

- **Ejemplo de Interacción:**
  1.  **Cliente -> Servidor:** `BET[{"agency":"1","first_name":"Luciano",...}]`
  2.  **Servidor -> Cliente:** `ACK[1]`
