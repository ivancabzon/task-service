# Task Service API

REST API сервис для управления задачами внутри команд. Включает ролевую модель (owner/admin/member), историю изменений, кеширование списка задач в Redis и аналитику на основе сложного SQL-запроса.

## 🛠 Стек технологий

*   **Язык:** Go 1.26
*   **БД:** MySQL 8.0
*   **Кеш:** Redis 7
*   **Фреймворки:** Gin (HTTP), sqlx (SQL), golang-jwt, go-redis
*   **Инфраструктура:** Docker, Docker Compose
*   **Документация:** Swagger (OpenAPI 2.0)

## 🚀 Быстрый старт

Проект полностью контейнеризирован. Для запуска потребуется только установленный Docker.

```bash
# 1. Склонируйте репозиторий
git clone https://github.com/ivancabzon/task-service.git
cd task-service

# 2. Создайте файл конфигурации (по умолчанию настроен на Docker)
cp .env.example .env

# 3. Запустите приложение, БД и Redis
docker compose up -d --build

# 4. Проверьте статус
curl http://localhost:8080/health
# Ожидаемый ответ: {"status":"ok"}
```

Миграции базы данных (создание таблиц) применяются автоматически при первом запуске контейнера MySQL благодаря механизму `docker-entrypoint-initdb.d`.

## 📖 Документация API (Swagger)

После запуска проекта интерактивная документация доступна по адресу:
👉 **http://localhost:8080/swagger/index.html**

**Как тестировать защищенные эндпоинты:**
1. Вызовите `POST /api/v1/register` и `POST /api/v1/login`.
2. Скопируйте `token` из ответа.
3. Нажмите кнопку **Authorize** 🔒 в правом верхнем углу Swagger UI.
4. Введите `Bearer <ваш_токен>` (обязательно с префиксом `Bearer ` и пробелом).

## 🧪 Тестирование

В проекте реализован обязательный интеграционный тест на SQL-отчет (`GET /teams/:id/stats`), который проверяет корректность агрегатов, работу с датами и изоляцию данных между командами.

Поскольку тест требует реального подключения к БД, он изолирован с помощью build-тега `//go:build integration` (чтобы обычные юнит-тесты запускались мгновенно и без Docker).

### Как запустить тест:

**1. Поднимите инфраструктуру (если еще не запущена):**
```bash
docker compose up -d
```

**2. Создайте изолированную тестовую БД:**
```bash
docker compose exec db mysql -uroot -ppassword -e "CREATE DATABASE IF NOT EXISTS taskdb_test;"
```

**3. Задайте переменную окружения (выберите свою ОС):**

*Для Linux / macOS (Bash/Zsh) или Windows (Git Bash):*
```bash
export TEST_DB_DSN="root:password@tcp(localhost:3306)/taskdb_test?parseTime=true&multiStatements=true"
```

*Для Windows (PowerShell):*
```powershell
$env:TEST_DB_DSN = "root:password@tcp(localhost:3306)/taskdb_test?parseTime=true&multiStatements=true"
```

*Для Windows (CMD):*
```cmd
set TEST_DB_DSN=root:password@tcp(localhost:3306)/taskdb_test?parseTime=true&multiStatements=true
```

**4. Запустите тест:**
```bash
go test -tags=integration ./internal/repository/ -run TestStatsRepo_Integration -v
```

*Ожидаемый результат: `--- PASS: TestStatsRepo_Integration`.*