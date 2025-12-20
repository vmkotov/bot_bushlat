#!/bin/bash

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🚀 Начинаем деплой Bushlatinga Bot v2.0 в Yandex Cloud...${NC}"

# Проверяем наличие Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker не установлен. Установите Docker:${NC}"
    echo "https://docs.docker.com/get-docker/"
    exit 1
fi

# Проверяем наличие yc CLI
if ! command -v yc &> /dev/null; then
    echo -e "${RED}❌ Yandex Cloud CLI не установлен. Установите:${NC}"
    curl -sSL https://storage.yandexcloud.net/yandexcloud-yc/install.sh | bash
    echo -e "${YELLOW}⚠️  Теперь выполните:${NC}"
    echo "yc init"
    exit 1
fi

# Проверяем авторизацию в YC
if ! yc config list &> /dev/null; then
    echo -e "${YELLOW}⚠️  Вы не авторизованы в Yandex Cloud. Запустите:${NC}"
    echo "yc init"
    exit 1
fi

echo -e "${GREEN}✅ Все проверки пройдены${NC}"

# Запрашиваем ID реестра
read -p "Введите ID вашего Container Registry (например, crp9tqoau5p3b0oq9g): " REGISTRY_ID
if [ -z "$REGISTRY_ID" ]; then
    echo -e "${RED}❌ ID реестра не может быть пустым${NC}"
    exit 1
fi

echo -e "${YELLOW}📝 Заполните переменные окружения для бота:${NC}"

# Запрашиваем обязательные переменные
read -p "Введите TELEGRAM_BOT_TOKEN: " TELEGRAM_BOT_TOKEN
if [ -z "$TELEGRAM_BOT_TOKEN" ]; then
    echo -e "${RED}❌ TELEGRAM_BOT_TOKEN не может быть пустым${NC}"
    exit 1
fi

read -p "Введите DATABASE_URL (строка подключения к Supabase): " DATABASE_URL
if [ -z "$DATABASE_URL" ]; then
    echo -e "${YELLOW}⚠️  DATABASE_URL не указан, бот будет работать в memory-only режиме${NC}"
fi

read -p "Введите ADMIN_CHAT_ID (ваш ID в Telegram): " ADMIN_CHAT_ID
if [ -z "$ADMIN_CHAT_ID" ]; then
    ADMIN_CHAT_ID="266468924"
    echo -e "${YELLOW}⚠️  Используем ADMIN_CHAT_ID по умолчанию: 266468924${NC}"
fi

# Опциональные переменные
read -p "Введите DEBUG (true/false, по умолчанию false): " DEBUG
DEBUG=${DEBUG:-false}

read -p "Введите LOG_LEVEL (info/debug/error, по умолчанию info): " LOG_LEVEL
LOG_LEVEL=${LOG_LEVEL:-info}

# 🔧 НОВОЕ: Запрашиваем Service Account ID
echo -e "${YELLOW}🔐 Настройка Service Account для Yandex Cloud...${NC}"
read -p "Введите SERVICE_ACCOUNT_ID (или оставьте пустым для автоматического создания): " SERVICE_ACCOUNT_ID

if [ -z "$SERVICE_ACCOUNT_ID" ]; then
    echo "📝 Создаем новый Service Account..."
    SA_NAME="bushlatinga-sa-$(date +%Y%m%d-%H%M%S)"
    
    # Создаем Service Account
    if ! yc iam service-account create --name "$SA_NAME" --description "Service Account для Bushlatinga Bot"; then
        echo -e "${RED}❌ Ошибка создания Service Account${NC}"
        exit 1
    fi
    
    # Получаем ID созданного SA
    SERVICE_ACCOUNT_ID=$(yc iam service-account get --name "$SA_NAME" --format json | jq -r '.id' 2>/dev/null)
    
    if [ -z "$SERVICE_ACCOUNT_ID" ]; then
        echo -e "${RED}❌ Не удалось получить ID созданного Service Account${NC}"
        echo -e "${YELLOW}⚠️  Попробуйте создать SA вручную:${NC}"
        echo "yc iam service-account create --name bushlatinga-sa"
        echo "yc iam service-account list"
        exit 1
    fi
    
    echo -e "${GREEN}✅ Создан Service Account: $SA_NAME (ID: $SERVICE_ACCOUNT_ID)${NC}"
else
    echo -e "${GREEN}✅ Используем существующий Service Account: $SERVICE_ACCOUNT_ID${NC}"
fi

# Формируем полный список переменных окружения
ENV_VARS="TELEGRAM_BOT_TOKEN=$TELEGRAM_BOT_TOKEN"
ENV_VARS="$ENV_VARS,ADMIN_CHAT_ID=$ADMIN_CHAT_ID"
ENV_VARS="$ENV_VARS,DEBUG=$DEBUG"
ENV_VARS="$ENV_VARS,LOG_LEVEL=$LOG_LEVEL"

if [ -n "$DATABASE_URL" ]; then
    ENV_VARS="$ENV_VARS,DATABASE_URL=$DATABASE_URL"
    echo -e "${GREEN}✅ Бот будет работать с Supabase PostgreSQL${NC}"
else
    echo -e "${YELLOW}⚠️  Бот будет работать в memory-only режиме (без БД)${NC}"
fi

echo -e "${YELLOW}🔨 Сборка Docker образа...${NC}"
docker build -t cr.yandex/$REGISTRY_ID/bushlatinga-bot:latest -f Dockerfile.yc .

# Авторизация в Container Registry
echo -e "${YELLOW}🔑 Авторизация в Container Registry...${NC}"
if ! yc container registry configure-docker; then
    echo -e "${RED}❌ Ошибка авторизации в Container Registry${NC}"
    echo -e "${YELLOW}⚠️  Попробуйте вручную:${NC}"
    echo "yc iam create-token | docker login --username iam --password-stdin cr.yandex"
    exit 1
fi

# Загрузка образа в реестр
echo -e "${YELLOW}📦 Загрузка образа в Container Registry...${NC}"
docker push cr.yandex/$REGISTRY_ID/bushlatinga-bot:latest

# Создание Serverless Container (если не существует)
echo -e "${YELLOW}🚀 Создание/обновление Serverless Container...${NC}"
if ! yc serverless container get --name bushlatinga-bot &> /dev/null; then
    echo -e "${YELLOW}📝 Создаем новый контейнер...${NC}"
    if ! yc serverless container create --name bushlatinga-bot; then
        echo -e "${RED}❌ Ошибка создания контейнера${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ Контейнер создан${NC}"
else
    echo -e "${YELLOW}⚠️  Контейнер уже существует, обновляем...${NC}"
fi

# Создание новой ревизии контейнера
echo -e "${YELLOW}⚙️  Создание новой ревизии контейнера...${NC}"
if ! yc serverless container revision deploy \
    --container-name bushlatinga-bot \
    --image cr.yandex/$REGISTRY_ID/bushlatinga-bot:latest \
    --cores 1 \
    --memory 128MB \
    --concurrency 1 \
    --execution-timeout 300s \
    --service-account-id "$SERVICE_ACCOUNT_ID" \
    --environment "$ENV_VARS"; then
    echo -e "${RED}❌ Ошибка деплоя ревизии${NC}"
    exit 1
fi

echo -e "${GREEN}🎉 Деплой завершён успешно!${NC}"
echo -e "${YELLOW}📋 Информация о развертывании:${NC}"
echo "• Реестр: cr.yandex/$REGISTRY_ID"
echo "• Образ: bushlatinga-bot:latest"
echo "• Контейнер: bushlatinga-bot"
echo "• Service Account: $SERVICE_ACCOUNT_ID"
echo "• Память: 128MB"
echo "• Таймаут: 300s"
echo "• Переменные окружения:"
echo "  - TELEGRAM_BOT_TOKEN: ✅ установлен"
echo "  - ADMIN_CHAT_ID: $ADMIN_CHAT_ID"
if [ -n "$DATABASE_URL" ]; then
    echo "  - DATABASE_URL: ✅ установлен (Supabase)"
else
    echo "  - DATABASE_URL: ❌ не установлен (memory-only)"
fi
echo "  - DEBUG: $DEBUG"
echo "  - LOG_LEVEL: $LOG_LEVEL"

echo -e "${YELLOW}📊 Проверьте статус:${NC}"
yc serverless container revision list --container-name bushlatinga-bot