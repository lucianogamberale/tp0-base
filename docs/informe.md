# Informe de Resolución - TP0

Este documento detalla la estrategia de resolución para cada ejercicio del Trabajo Práctico N°0, junto con las decisiones de diseño clave sobre el protocolo de comunicación y los mecanismos de concurrencia implementados.

---

## Ejecución y Detalles de la Solución

A continuación se detalla la estrategia de resolución para cada ejercicio, junto con las instrucciones para ejecutar y verificar el funcionamiento de cada etapa.

Cada ejercicio se encuentra en su propia rama de Git. Para probar una solución específica, primero debe posicionarse en la rama correspondiente, por ejemplo: `git checkout ej1`.

---

### Ejercicio 1: Generar Docker Compose configurable

- **Objetivo:** Crear un script `generar-compose.sh` que genere un archivo `docker-compose.yaml` con una cantidad configurable de clientes.

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

- **Objetivo:** Modificar el sistema para que los cambios en los archivos de configuración (`config.ini` para el servidor y `config.yaml` para el cliente) no requieran reconstruir las imágenes de Docker.

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

### Ejercicio 6: Procesamiento por Lotes (Batches)

- **Objetivo:** Modificar el cliente para que lea las apuestas desde un archivo `.csv`, las agrupe en lotes (_batches_) de tamaño configurable y las envíe en una única transacción al servidor. El tamaño de cada lote está limitado tanto por cantidad de apuestas como por tamaño total en KiB.

- **Implementación (Cliente - Go):**
  El cliente fue reestructurado para funcionar como un procesador de archivos, leyendo datos de un `.csv` y enviándolos eficientemente en lotes.

  1.  **Ingestión de Datos desde CSV:** La lógica ahora se centra en leer el archivo `agency-N.csv` correspondiente a cada cliente. Se utiliza el paquete `encoding/csv` de Go para parsear el archivo de manera robusta. El archivo se inyecta en el contenedor mediante **`docker volumes`** y leido de a lotes.

  2.  **Lógica de Creación de Lotes:** El corazón de esta implementación es la función `readBetBatchFromCsvUsing`. Al construir un lote, se aplica una **doble condición de corte** para asegurar la eficiencia y el cumplimiento de los límites:

      - **Límite de Cantidad:** El lote no puede superar el `MaxAmountOfBetsOnEachBatch` definido en la configuración.
      - **Límite de Tamaño:** Antes de añadir una nueva apuesta al lote, se calcula el tamaño que tendría el mensaje final serializado y se verifica que no exceda el `MaxKiBPerBatch` configurado.

  3.  **Serialización de Lotes:** Se modificó el protocolo para que el `PAYLOAD` de un mensaje `BET` pueda contener múltiples apuestas. La función `EncodeBetBatchMessage` se encarga de serializar cada apuesta del lote y unirlas con un separador (`;`).

  4.  **ACK por Lote:** El mecanismo de confirmación (`ACK`) fue mejorado. Ahora, tras enviar un lote, el cliente espera que el servidor responda con un `ACK` cuyo `PAYLOAD` sea la **cantidad de apuestas procesadas** en ese lote (ej: `ACK[50]`).

  5.  **Notificación de Finalización:** Se introdujo un nuevo tipo de mensaje, `NMB` (No More Bets). Una vez que el cliente ha leído y enviado todas las apuestas de su archivo `.csv`, envía este mensaje final al servidor para notificar que ha completado su trabajo.

- **Implementación (Servidor - Python):**
  El servidor fue adaptado para procesar lotes de apuestas, manteniendo una conexión persistente con cada cliente durante toda su sesión de envío.

  1.  **Conexiones Persistentes:** A diferencia del ejercicio anterior, el servidor ahora mantiene la conexión con un cliente activa. El método `__handle_client_connection` fue refactorizado con un bucle `while` que procesa múltiples mensajes de un mismo cliente hasta recibir la señal de finalización.

  2.  **Enrutamiento de Mensajes:** Dentro del bucle de conexión, el servidor actúa como un enrutador simple. Inspecciona el tipo de mensaje (`BET` o `NMB`) y lo delega a la función manejadora correspondiente (`__handle_bet_batch_message` o `__handle_no_more_bets_message`).

  3.  **Deserialización de Lotes:** Al recibir un mensaje `BET`, el servidor utiliza la función `decode_bet_batch_message` para deserializar el `PAYLOAD`. Esta función divide el `PAYLOAD` por el separador (`;`) y reconstruye una lista de objetos `Bet`.

  4.  **Almacenamiento y Confirmación de Lotes:** El servidor invoca `utils.store_bets()` una sola vez con la lista completa de apuestas del lote. Luego, envía al cliente el `ACK` correspondiente, conteniendo la cantidad de apuestas procesadas (ej: `ACK[50]`). En caso de error, responde con `ACK[0]`.

  5.  **Manejo de Finalización:** Al recibir el mensaje `NMB`, el servidor responde con un `ACK[NMB]` y aumenta su contador interno de agencias finalizadas. Este contador le permite saber cuándo todas las agencias han terminado para poder cerrar su propia ejecución.

- **Ejecución:**
  1.  Posicionarse en la rama: `git checkout ej6`.
  2.  El `docker-compose.yaml` (generado por el script) debe usar **volúmenes** para mapear los archivos `.data/agency-{N}.csv` al interior de cada contenedor de cliente.
  3.  Levantar los servicios: `make docker-compose-up`.
  4.  En los logs (`make docker-compose-logs`) se observará cómo los clientes leen y envían lotes de apuestas, y el servidor confirma cada lote con el log `action: apuesta_recibida`. Finalmente, los clientes enviarán la notificación de finalización y el servidor se apagará.

### Ejercicio 7: Sorteo y Consulta de Ganadores

- **Objetivo:** Implementar la fase final del sorteo. Los clientes, tras enviar todas sus apuestas, deben notificar al servidor y luego consultarle periódicamente por los ganadores de su agencia. El servidor solo podrá responder con los resultados una vez que **todas** las agencias hayan finalizado su envío.

- **Implementación (Cliente - Go):**
  El cliente ahora opera en un flujo de dos fases claramente diferenciadas: una de envío y otra de consulta.

  1.  **Fase 1 - Envío de Apuestas:** Esta fase es idéntica a la del ejercicio 6. El cliente se conecta, envía todas sus apuestas en lotes y finaliza con un mensaje `NMB` (No More Bets) para notificar que ha terminado. Todo esto ocurre sobre una única conexión persistente.

  2.  **Fase 2 - Consulta de Ganadores (Polling):** Una vez finalizada la fase de envío, el cliente entra en un bucle de sondeo (`polling`) para consultar los resultados.
      - **Nueva Lógica de Conexión:** Para esta fase, el cliente adopta una estrategia de conexiones cortas: establece una nueva conexión para cada consulta y la cierra inmediatamente después.
      - **Nuevos Mensajes del Protocolo:** El cliente envía un nuevo tipo de mensaje, `ASK`, para solicitar los ganadores. El servidor puede responder de dos maneras:
        - **`WIT` (Wait):** Si el sorteo aún no se ha realizado, el servidor responde con este mensaje.
        - **`WIN` (Winners):** Si el sorteo ya ocurrió, el servidor responde con la lista de DNI ganadores.
      - **Manejo de Respuestas:** El cliente actúa según la respuesta recibida:
        - Si recibe `WIT`, interpreta que debe esperar. El cliente hace una pausa (`time.Sleep`) durante un período configurable (`WaitLoopPeriod`) y vuelve a intentarlo.
        - Si recibe `WIN`, el bucle de sondeo termina. El cliente decodifica el `PAYLOAD` para obtener la lista de ganadores, imprime en el log el mensaje final `action: consulta_ganadores` con la cantidad, y finaliza su ejecución.

- **Implementación (Servidor - Python):**
  El servidor tiene ahora una lógica de estados para gestionar el proceso del sorteo de forma sincronizada.

  1.  **Máquina de Estados:** El servidor ahora funciona como una máquina de estados simple, controlada por el contador `_number_of_finished_agencies`. Pasa del estado "recibiendo apuestas" al estado "sorteo realizado".

  2.  **Disparador del Sorteo:** El sorteo se considera realizado (`__was_draw_held()` devuelve `True`) en el momento exacto en que el servidor recibe el **quinto y último** mensaje `NMB`. En ese instante, emite el log `action: sorteo | result: success`.

  3.  **Gestión de Consultas:** El servidor ahora puede manejar mensajes `ASK`. Su respuesta depende del estado actual:

      - **Antes del Sorteo:** Si recibe un `ASK` pero `__was_draw_held()` es `False`, responde con un mensaje `WIT`, indicando al cliente que debe esperar.
      - **Después del Sorteo:** Si recibe un `ASK` y el sorteo ya se realizó, el servidor utiliza las funciones `utils.load_bets()` y `utils.has_won()` para filtrar y encontrar los ganadores **específicos de la agencia que realizó la consulta**. Luego, serializa los DNIs ganadores en un mensaje `WIN` y lo envía.

  4.  **Condición de Apagado:** El servidor ahora espera no solo a que se realice el sorteo, sino a que todas las agencias hayan consultado y recibido a sus ganadores (`__all_agencies_with_winners()`) antes de finalizar su ejecución.

- **Ejecución:**
  1.  Posicionarse en la rama: `git checkout ej7`.
  2.  Levantar los servicios: `make docker-compose-up`.
  3.  En los logs (`make docker-compose-logs`), se observará el siguiente flujo:
      - Todas las agencias envían sus lotes y sus mensajes `NMB`.
      - Tras recibir el último `NMB`, el servidor imprime `action: sorteo | result: success`.
      - Inmediatamente, los clientes comienzan a enviar mensajes `ASK`. El servidor responde con mensajes `WIN`.
      - Cada cliente, al recibir su lista, imprime `action: consulta_ganadores | result: success`.
      - Una vez que todos los clientes han recibido sus resultados, el servidor se apaga.

### Ejercicio 8: Concurrencia en el Servidor

- **Objetivo:** Modificar el servidor para que pueda aceptar y procesar conexiones de múltiples clientes en paralelo. Esto requiere el uso de hilos (_threads_) y la implementación de mecanismos de sincronización para garantizar la consistencia de los datos compartidos y coordinar el evento del sorteo.

- **Implementación (Cliente - Go):**
  Para adaptarse al nuevo comportamiento del servidor, el cliente fue **simplificado**, eliminando la lógica de sondeo (_polling_), dado que como el server acepta conexiones en paralelo, el cliente podía mantener una conexión persistente sin afectar a los demás.

  1.  **Comunicación única:** El flujo de comunicación del cliente ahora es completamente secuencial y ocurre en una única conexión persistente. Las fases de envío y consulta de ganadores ya no están separadas por conexiones diferentes.

  2.  **Eliminación del Polling:** Se eliminó el bucle `while` que realizaba consultas periódicas. Tras enviar el mensaje `NMB`, el cliente **inmediatamente envía el mensaje `ASK` por la misma conexión**.

  3.  **Espera Bloqueante:** El cliente ahora se bloquea esperando la respuesta `WIN` después de enviar su `ASK`. Como el servidor concurrente puede manejar la espera internamente, ya no es necesario el mensaje `WIT` (Wait). El cliente asume que la respuesta a su `ASK` será la definitiva.

  Este cambio hace al cliente más simple y eficiente, delegando la complejidad de la sincronización del sorteo completamente al servidor.

- **Implementación (Servidor - Python):**
  El servidor fue rediseñado para adoptar un modelo concurrente **"thread-per-client"**, utilizando primitivas de sincronización del módulo `threading` para gestionar el sorteo. Es importante destacar que a pesar de las [limitaciones propias del lenguaje](https://wiki.python.org/moin/GlobalInterpreterLock), se cumple el paralelismo solicitado por no ser

  1.  **Arquitectura "Thread-Per-Client":** El hilo principal del servidor ahora tiene una única responsabilidad: aceptar nuevas conexiones en un bucle. Por cada conexión aceptada, instancia y lanza un nuevo hilo de ejecución (`threading.Thread`) que se encargará de gestionar toda la comunicación con ese cliente específico. Esto permite que las cinco agencias envíen sus apuestas y consulten resultados de forma paralela.

  2.  **Sincronización con `threading.Barrier`:** El mecanismo central para coordinar el sorteo es una **barrera**.

      - Se inicializa una barrera (`threading.Barrier`) con el número total de agencias (5).
      - Cada hilo de cliente, después de procesar todos los lotes y recibir el mensaje `ASK`, llega a la línea `self._draw_barrier.wait()`.
      - En este punto, el hilo se bloquea. La barrera lleva un conteo interno de cuántos hilos han llegado.
      - Cuando el **quinto y último hilo** llega y llama a `.wait()`, la barrera se "rompe", y todos los hilos que estaban esperando son liberados **simultáneamente** para continuar su ejecución.
      - El hilo que rompe la barrera es el encargado de imprimir el log `action: sorteo | result: success`, asegurando que se imprima una sola vez en el momento justo.

  3.  **Gestión de Hilos:** El hilo principal, después de lanzar todos los hilos de los clientes, espera a que todos terminen su ejecución (`thread.join()`) antes de finalizar el programa, garantizando un apagado limpio.

- **Ejecución:**
  1.  Posicionarse en la rama: `git checkout ej8`.
  2.  Levantar los servicios: `make docker-compose-up`.
  3.  Al revisar los logs (`make docker-compose-logs`), se podrá observar el comportamiento concurrente:
      - Los logs de `apuesta_recibida` de diferentes clientes aparecerán **intercalados**, demostrando el procesamiento en paralelo.
      - Aparecerá el único log de `action: sorteo | result: success`.
      - Inmediatamente después, aparecerán los cinco logs de `action: consulta_ganadores` de los clientes casi al mismo tiempo, evidenciando que todos fueron liberados por la barrera de forma simultánea.

---

## Aspectos de Diseño

### Protocolo de Comunicación

El protocolo se simplifica en su fase final, eliminando la necesidad de un mensaje de espera.

- **Formato General del Mensaje:**
  `TIPO[PAYLOAD]`

- **Formato General del Mensaje:**
  `TIPO[PAYLOAD]`

- **Tipos de Mensajes (Ejercicio 5):**

  - **`BET`**: Para una única apuesta. `PAYLOAD`: `{"campo":"valor",...}`.
  - **`ACK`**: Confirmación simple. `PAYLOAD`: `1`.

- **Evolución para el Ejercicio 6:**

  - **`NMB` (No More Bets):** El cliente notifica el fin del envío. `PAYLOAD`: `{"agency":"1"}`.
  - **`BET` (Modificado):** `PAYLOAD` ahora contiene lotes. `BET[{...};{...}]`.
  - **`ACK` (Modificado):** `PAYLOAD` contiene la cantidad de apuestas en un lote (`ACK[50]`) o confirma el `NMB` (`ACK[NMB]`).

- **Evolución para el Ejercicio 7:**

  - **`ASK` (Ask for Winners):**
    - **Propósito:** Enviado por el cliente para solicitar la lista de ganadores de su agencia.
    - **Payload:** Identifica a la agencia que consulta. `{"agency":"1"}`.
  - **`WIT` (Wait):**
    - **Propósito:** Enviado por el servidor para indicar que el sorteo aún no se ha realizado y el cliente debe esperar.
    - **Payload:** Vacío.
  - **`WIN` (Winners):**
    - **Propósito:** Enviado por el servidor con la lista final de ganadores para una agencia.
    - **Payload:** Una lista de DNI ganadores, separados por comas. `"12345678","87654321"`.

- **Evolución para el Ejercicio 8:**

  - **Mensaje Eliminado - `WIT` (Wait):**
    - Este mensaje ya no es necesario. El servidor concurrente gestionará la espera de los clientes internamente, bloqueando la conexión hasta que el sorteo se realice. La ausencia de este mensaje simplifica la lógica del cliente.

- **Ejemplo de Interacción Final:**
  1.  **Cliente -> Servidor (Lote 1):** `BET[{"doc":"111",...};{"doc":"222",...}]`
  2.  **Servidor -> Cliente:** `ACK[2]`
  3.  _(... más lotes ...)_
  4.  **Cliente -> Servidor (Fin Lotes):** `NMB[{"agency":"1"}]`
  5.  **Servidor -> Cliente:** `ACK[NMB]`
  6.  **Cliente -> Servidor (Consulta):** `ASK[{"agency":"1"}]`
  7.  _(El cliente se bloquea en la lectura, esperando la respuesta)_
  8.  _(El servidor espera a que todas las agencias lleguen a la barrera, realiza el sorteo y finalmente responde)_
  9.  **Servidor -> Cliente:** `WIN["11122233","44455566"]`

## Mecanismos de Sincronización

Para garantizar el correcto funcionamiento del servidor en un entorno concurrente, se utilizaron las siguientes primitivas del módulo `threading` de Python.

### Elección de Threads vs. Procesos: El Rol del GIL

Una decisión fundamental en la concurrencia con Python es elegir entre `threading` y `multiprocessing`. Aunque a primera vista el **Global Interpreter Lock (GIL)** parece invalidar el uso de hilos para el paralelismo real, la naturaleza de nuestra aplicación hace que `threading` sea la opción ideal.

- **¿Qué es el GIL?** El GIL es un mutex que protege los objetos de Python, asegurando que solo **un hilo ejecute bytecode de Python a la vez** en un único proceso. Esto significa que para tareas **CPU-bound** (cálculos intensivos), los hilos no pueden aprovechar múltiples núcleos de CPU y `multiprocessing` sería la elección correcta.

- **Nuestra Aplicación es I/O-bound:** Nuestro servidor no realiza cálculos complejos. Su principal actividad es esperar: esperar por nuevas conexiones (`socket.accept()`), esperar a que los clientes envíen datos (`socket.recv()`) y esperar a que los datos se envíen por la red (`socket.sendall()`). Este tipo de carga de trabajo se denomina **I/O-bound** (limitada por la Entrada/Salida).

- **¿Por qué Threads?** La clave es que el GIL **se libera** cuando un hilo inicia una operación de I/O bloqueante. Mientras un hilo está pasivamente esperando datos de la red, el GIL está disponible, permitiendo que otro hilo se ejecute. Esto crea un modelo de **concurrencia muy eficiente**: mientras el Hilo A espera por el Lote 2 del Cliente 1, el Hilo B puede estar procesando el Lote 1 del Cliente 2. Aunque no hay paralelismo real de ejecución de Python, el tiempo de espera de I/O se solapa, dando como resultado un alto rendimiento.

En resumen, para nuestra aplicación que maneja muchas operaciones IO, los hilos son la herramienta correcta porque permiten manejar de manera mas "liviana" conexiones simultáneas eficientemente, aprovechando los tiempos muertos de espera de red.

### Primitivas Utilizadas

- **`threading.Thread`:**
  Se adopta el modelo **"thread-per-client"**, donde cada cliente conectado es atendido por un hilo de ejecución independiente. Esto permite que el procesamiento de las apuestas de múltiples agencias ocurra en paralelo, mejorando significativamente el rendimiento y la capacidad de respuesta del sistema.

- **`threading.Barrier`:**
  Es la herramienta de sincronización clave para el sorteo. En este caso, se inicializa con el número de agencias. Cada hilo de cliente se bloquea en la barrera (`barrier.wait()`) al momento de consultar por los ganadores. Ningún hilo puede avanzar más allá de este punto hasta que **todos** los hilos hayan llegado. Esto garantiza por diseño que el sorteo no se procese hasta que la última agencia haya finalizado su envío y esté lista para recibir los resultados, resolviendo el problema de sincronización de una manera elegante y eficiente.

- **`threading.Event`:**
  Se utiliza como un _flag_ booleano seguro para hilos (_thread-safe_). En esta implementación, se usa para gestionar el estado de ejecución del servidor (`_server_running`). El hilo principal verifica este evento para continuar aceptando conexiones, y el manejador de `SIGTERM` lo utiliza para señalar a todos los hilos que deben finalizar su ejecución de forma ordenada.
