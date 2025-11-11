# Booking Platform - Python + WhatsApp Edition 🐍📱

Полнофункциональная платформа для бронирования услуг с интеграцией WhatsApp вместо email уведомлений. Построена на Python (FastAPI) и Node.js (для WhatsApp).

## 🚀 Основные возможности

- **Multi-tenant архитектура**: Поддержка множества бизнесов с поддоменами
- **WhatsApp уведомления**: Автоматическая отправка уведомлений через WhatsApp
- **Микросервисная архитектура**: 6 независимых Python сервисов + WhatsApp сервис на Node.js
- **Управление бронированиями**: Полный цикл бронирования от создания до завершения
- **Роли пользователей**: SUPER_ADMIN, OWNER, MANAGER, MASTER, CLIENT
- **Многоязычность**: Поддержка русского, английского, казахского
- **Background tasks**: Celery для фоновых задач и напоминаний

## 🏗 Архитектура

### Микросервисы

1. **API Gateway** (порт 8000) - Главная точка входа, маршрутизация, аутентификация
2. **User Service** (порт 8001) - Управление пользователями и аутентификация
3. **Booking Service** (порт 8002) - Логика бронирования и проверка доступности
4. **Notification Service** (порт 8003) - Отправка уведомлений через WhatsApp
5. **Payment Service** (порт 8004) - Обработка платежей (заглушка)
6. **Admin Service** (порт 8005) - Администрирование платформы
7. **WhatsApp Service** (порт 3000) - Node.js сервис для WhatsApp интеграции

### Технологический стек

- **Backend**: Python 3.11 + FastAPI
- **WhatsApp**: Node.js 18 + whatsapp-web.js
- **Database**: PostgreSQL 15
- **Cache**: Redis 7
- **Background Jobs**: Celery
- **Containerization**: Docker & Docker Compose

## 📋 Предварительные требования

- Docker 20.10+
- Docker Compose 2.0+
- 4GB RAM минимум
- WhatsApp аккаунт для отправки сообщений

## 🚀 Быстрый старт

### 1. Клонирование и настройка

```bash
# Создать .env файл из примера
cp .env.example .env

# Отредактировать .env и установить необходимые значения
nano .env
```

### 2. Запуск всех сервисов

```bash
# Собрать и запустить все контейнеры
docker-compose up --build -d

# Проверить статус сервисов
docker-compose ps
```

### 3. Инициализация базы данных

```bash
# Создать таблицы и начальные данные
docker-compose exec api-gateway python migrations/init_db.py
```

### 4. Настройка WhatsApp

```bash
# Получить QR код для аутентификации WhatsApp
curl http://localhost:3000/qr

# ИЛИ просмотреть логи для QR кода в терминале
docker-compose logs -f whatsapp-service
```

Отсканируйте QR код в WhatsApp на телефоне:
1. Откройте WhatsApp
2. Перейдите в Настройки → Связанные устройства
3. Нажмите "Связать устройство"
4. Отсканируйте QR код из терминала

### 5. Проверка работоспособности

```bash
# API Gateway
curl http://localhost:8000/health

# WhatsApp Service
curl http://localhost:3000/health

# Документация API
открыть http://localhost:8000/api/docs
```

## 📱 API Документация

### Swagger UI
Интерактивная документация доступна по адресу:
```
http://localhost:8000/api/docs
```

### Основные эндпоинты

#### Аутентификация
```bash
# Регистрация нового бизнеса
POST /api/v1/register
{
  "email": "owner@business.com",
  "password": "securepass123",
  "full_name": "Иван Иванов",
  "phone": "+77771234567",
  "business_name": "Мой салон красоты",
  "subdomain": "mysalon"
}

# Вход
POST /api/v1/login
{
  "email": "owner@business.com",
  "password": "securepass123"
}
```

#### Публичные эндпоинты (для клиентов)
```bash
# Получить информацию о бизнесе
GET /api/v1/public/business/{subdomain}

# Получить услуги
GET /api/v1/public/business/{subdomain}/services

# Получить мастеров
GET /api/v1/public/business/{subdomain}/masters

# Проверить доступность
GET /api/v1/public/business/{subdomain}/availability?master_id=1&date=2024-01-15

# Создать бронирование
POST /api/v1/public/booking
{
  "subdomain": "mysalon",
  "client_phone": "+77779876543",
  "client_name": "Алия Сарсенова",
  "master_id": 1,
  "service_id": 1,
  "booking_date": "2024-01-15T14:00:00",
  "notes": "Предпочитаю окно"
}
```

#### Защищенные эндпоинты
```bash
# Все запросы требуют заголовок:
Authorization: Bearer <access_token>

# Получить мои бронирования
GET /api/v1/bookings

# Отменить бронирование
DELETE /api/v1/booking/{booking_id}
```

## 📨 WhatsApp интеграция

### Отправка сообщений

```bash
# Отправить сообщение
curl -X POST http://localhost:3000/send-message \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "+77771234567",
    "message": "Привет! Это тестовое сообщение"
  }'

# Проверить номер
curl -X POST http://localhost:3000/check-number \
  -H "Content-Type: application/json" \
  -d '{"phone": "+77771234567"}'

# Массовая рассылка
curl -X POST http://localhost:3000/send-bulk \
  -H "Content-Type: application/json" \
  -d '{
    "phones": ["+77771234567", "+77779876543"],
    "message": "Акция! Скидка 20% на все услуги"
  }'
```

### Формат телефонных номеров

WhatsApp Service автоматически форматирует номера:
- `87771234567` → `77771234567@c.us`
- `+77771234567` → `77771234567@c.us`
- `7771234567` → `77771234567@c.us`

## 🛠 Команды управления

### Docker Compose

```bash
# Запуск всех сервисов
docker-compose up -d

# Остановка всех сервисов
docker-compose down

# Перезапуск конкретного сервиса
docker-compose restart api-gateway

# Просмотр логов
docker-compose logs -f

# Просмотр логов конкретного сервиса
docker-compose logs -f whatsapp-service

# Пересборка сервисов
docker-compose up --build -d

# Очистка всех контейнеров и данных
docker-compose down -v
```

### База данных

```bash
# Инициализация БД
docker-compose exec api-gateway python migrations/init_db.py

# Подключиться к PostgreSQL
docker-compose exec postgres psql -U booking_user -d booking_platform

# Бэкап базы данных
docker-compose exec postgres pg_dump -U booking_user booking_platform > backup.sql

# Восстановление из бэкапа
docker-compose exec -T postgres psql -U booking_user booking_platform < backup.sql
```

### Redis

```bash
# Подключиться к Redis CLI
docker-compose exec redis redis-cli

# Очистить весь кеш
docker-compose exec redis redis-cli FLUSHALL

# Проверить количество ключей
docker-compose exec redis redis-cli DBSIZE
```

## 👥 Начальные учетные данные

После инициализации БД создаются следующие аккаунты:

### Super Admin
- **Email**: admin@jazyl.tech
- **Password**: admin123
- **Роль**: SUPER_ADMIN

### Demo бизнес
- **Subdomain**: demo
- **Owner Email**: owner@demo.jazyl.tech
- **Owner Password**: demo123

⚠️ **ВАЖНО**: Смените пароли в продакшене!

## 🔧 Конфигурация

Основные настройки в `.env`:

```env
# База данных
DATABASE_URL=postgresql://booking_user:booking_password@postgres:5432/booking_platform

# JWT
JWT_SECRET_KEY=your-secret-key-min-32-characters-long
ACCESS_TOKEN_EXPIRE_MINUTES=1440  # 24 часа
REFRESH_TOKEN_EXPIRE_DAYS=7

# WhatsApp
WHATSAPP_SERVICE_URL=http://whatsapp-service:3000
WHATSAPP_ENABLED=true

# Бизнес логика
DEFAULT_TRIAL_DAYS=30
CANCELLATION_HOURS=2
REMINDER_HOURS=24,2

# Язык
DEFAULT_LANGUAGE=ru
SUPPORTED_LANGUAGES=ru,en,kk
```

## 📊 Структура проекта

```
booking-platform/
├── api-gateway/           # API Gateway сервис
│   ├── main.py
│   ├── routes/           # HTTP маршруты
│   └── middleware/       # Аутентификация, rate limiting
├── user-service/         # Сервис пользователей
│   ├── main.py
│   └── services/
├── booking-service/      # Сервис бронирований
│   ├── main.py
│   └── services/
├── notification-service/ # Сервис уведомлений
│   └── main.py
├── admin-service/        # Админ панель
│   └── main.py
├── payment-service/      # Платежи (stub)
│   └── main.py
├── whatsapp-service/     # WhatsApp сервис (Node.js)
│   ├── package.json
│   ├── src/
│   │   └── server.js
│   └── Dockerfile
├── shared/               # Общие модули
│   ├── auth/            # JWT, хеширование
│   ├── cache/           # Redis клиент
│   ├── config/          # Конфигурация
│   ├── database/        # База данных
│   └── models/          # SQLAlchemy модели
├── migrations/           # Миграции БД
│   └── init_db.py
├── docker-compose.yml
├── Dockerfile.python
├── requirements.txt
├── .env.example
└── README.md
```

## 🔐 Безопасность

### Рекомендации

1. **Смените JWT_SECRET_KEY** в продакшене на случайную строку минимум 32 символа
2. **Смените пароли БД** в .env файле
3. **Используйте HTTPS** для API Gateway
4. **Ограничьте доступ** к портам сервисов (только API Gateway должен быть публичным)
5. **Регулярно обновляйте** зависимости

### Rate Limiting

API Gateway автоматически ограничивает:
- 100 запросов в минуту на IP адрес (настраивается в .env)

## 🐛 Устранение неполадок

### WhatsApp не подключается

```bash
# Проверить логи
docker-compose logs whatsapp-service

# Удалить сессию и переподключиться
docker-compose down
docker volume rm booking-platform_whatsapp_auth
docker-compose up -d whatsapp-service

# Получить новый QR код
curl http://localhost:3000/qr
```

### База данных не подключается

```bash
# Проверить что PostgreSQL запущен
docker-compose ps postgres

# Проверить логи
docker-compose logs postgres

# Пересоздать БД
docker-compose down -v
docker-compose up -d postgres
docker-compose exec api-gateway python migrations/init_db.py
```

### Сервисы не могут соединиться

```bash
# Проверить сеть Docker
docker network ls
docker network inspect booking-platform_booking-network

# Перезапустить все сервисы
docker-compose restart
```

## 📝 Разработка

### Локальная разработка

```bash
# Установить зависимости
pip install -r requirements.txt
cd whatsapp-service && npm install

# Запустить только БД и Redis
docker-compose up -d postgres redis

# Запустить сервисы локально
export $(cat .env | xargs)
python api-gateway/main.py  # порт 8000
python user-service/main.py # порт 8001
# и т.д.
```

### Тестирование

```bash
# Установить зависимости для тестов
pip install pytest pytest-asyncio httpx

# Запустить тесты
pytest
```

## 📄 Лицензия

Proprietary - Все права защищены

## 🤝 Поддержка

Для вопросов и проблем:
- Создайте Issue в репозитории
- Email: support@jazyl.tech

## 🎯 Roadmap

- [ ] Frontend приложение
- [ ] Mobile приложения (iOS/Android)
- [ ] Интеграция платежей (Kaspi, Paybox)
- [ ] SMS уведомления
- [ ] Email уведомления
- [ ] Telegram бот интеграция
- [ ] Аналитика и отчеты
- [ ] Экспорт данных
- [ ] API для интеграций
- [ ] Webhook система

---

**Версия**: 2.0.0 (Python Edition с WhatsApp)
**Обновлено**: 2024
