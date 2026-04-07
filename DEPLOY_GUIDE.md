# Руководство по деплою сервера OverSight

В этом документе описан процесс настройки чистого выделенного Linux-сервера и инструкция по его обновлению после каждого изменения в коде.

---

## 1. Первоначальная настройка сервера (Один раз)

Вам потребуется чистый сервер на Linux (рекомендуется **Ubuntu 22.04 / 24.04**). Зайдите на сервер по SSH и выполните следующие шаги:

### 1.1. Обновление системы и установка необходимых пакетов
```bash
sudo apt update && sudo apt upgrade -y
sudo apt install -y git curl apt-transport-https ca-certificates software-properties-common
```

### 1.2. Установка Docker и Docker Compose
Наш сервер запускается в изолированном контейнере, поэтому сам язык Go на сервер ставить **не нужно**. Всё соберётся внутри Docker.

```bash
# Добавляем ключ и репозиторий Docker
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# ВАЖНО: Обновляем списки пакетов ПОСЛЕ добавления репозитория Docker
sudo apt update

# Теперь устанавливаем Docker
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

# Добавляем текущего пользователя в группу docker (чтобы не писать sudo docker)
sudo usermod -aG docker $USER
```
*Примечание: после добавления в группу `docker`, перезайдите по SSH, чтобы изменения вступили в силу.*

### 1.3. Клонирование репозитория
Скачайте ваш проект на сервер:
```bash
# Клонируем проект из GitHub
git clone https://github.com/ReadixGG/oversight.git
cd oversight/server
```

---

## 2. Как деплоить обновления (После каждого изменения)

Когда вы внесли изменения в папку `server/` (например, добавили новую механику) и запушили их в Git, вам нужно обновить работающий сервер.

Выполните эти 3 простые команды на Linux-сервере:

```bash
# 1. Зайдите в папку с сервером
cd ~/oversight/server

# 2. Скачайте свежие изменения из Git
git pull origin main

# 3. Пересоберите и перезапустите сервер (без даунтайма старой версии до момента старта новой)
docker compose up -d --build
```

### Что делает команда `docker compose up -d --build`?
- `--build` заставляет Docker заново прочитать код и собрать новый образ сервера (внутри скачиваются пакеты Go и компилируется бинарник).
- `-d` (detach) запускает сервер в фоновом режиме, освобождая консоль.
- Старый контейнер будет остановлен и заменён новым только когда сборка успешно завершится.

---

## 3. Полезные команды для администрирования

### Просмотр логов сервера
Если нужно узнать, что происходит на сервере, кто подключился или почему произошла ошибка:
```bash
cd ~/oversight/server
docker compose logs -f
```
*(Для выхода из логов нажмите `Ctrl+C`)*

### Остановка сервера
Если нужно полностью выключить сервер:
```bash
cd ~/oversight/server
docker compose down
```

### Проверка статуса сервера (Health Check)
Сервер имеет встроенный эндпоинт для проверки того, что он живой:
```bash
curl http://localhost:8080/health
# Ожидаемый ответ: ok
```

---

## 4. Настройка Firewall (Опционально, но рекомендуется)

Чтобы обезопасить сервер, оставьте открытыми только порты для SSH (22) и игры (8080):
```bash
sudo ufw allow 22/tcp
sudo ufw allow 8080/tcp
sudo ufw enable
```
*(При запросе подтверждения активации фаервола ответьте `y`)*

---

## Итог (Краткая шпаргалка)

**Каждый раз при деплое вы просто пишете:**
```bash
cd ~/oversight/server && git pull && docker compose up -d --build
```
Это пересоберёт код и перезапустит игру за несколько секунд.
