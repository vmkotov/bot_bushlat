#!/bin/bash

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🚀 Начинаем деплой бота в Yandex Cloud...${NC}"

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

# Устанавливаем docker-credential-yc если его нет
echo -e "${YELLOW}🔧 Проверяем наличие docker-credential-yc...${NC}"
if ! command -v docker-credential-yc &> /dev/null; then
    echo -e "${YELLOW}⚠️  Устанавливаю docker-credential-yc...${NC}"
    
    # Определяем архитектуру системы
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    # Для macOS
    if [ "$OS" = "darwin" ]; then
        if [ "$ARCH" = "x86_64" ]; then
            BINARY="docker-credential-yc_darwin_amd64"
        elif [ "$ARCH" = "arm64" ]; then
            BINARY="docker-credential-yc_darwin_arm64"
        fi
    # Для Linux
    elif [ "$OS" = "linux" ]; then
        if [ "$ARCH" = "x86_64" ]; then
            BINARY="docker-credential-yc_linux_amd64"
        elif [ "$ARCH" = "aarch64" ]; then
            BINARY="docker-credential-yc_linux_arm64"
        fi
    fi
    
    if [ -z "$BINARY" ]; then
        echo -e "${RED}❌ Неподдерживаемая архитектура: $OS $ARCH${NC}"
        exit 1
    fi
    
    # Создаем директорию для плагинов Docker
    mkdir -p ~/.docker/cli-plugins
    
    # Скачиваем плагин
    DOWNLOAD_URL="https://github.com/yandex-cloud/docker-credential-yc/releases/latest/download/$BINARY"
    echo -e "${YELLOW}📥 Скачиваю: $DOWNLOAD_URL${NC}"
    
    if ! curl -L -o ~/.docker/cli-plugins/docker-credential-yc "$DOWNLOAD_URL"; then
        echo -e "${RED}❌ Ошибка скачивания docker-credential-yc${NC}"
        exit 1
    fi
    
    # Даем права на выполнение
    chmod +x ~/.docker/cli-plugins/docker-credential-yc
    
    # Добавляем в PATH для текущей сессии
    export PATH="$PATH:$HOME/.docker/cli-plugins"
    
    echo -e "${GREEN}✅ docker-credential-yc установлен${NC}"
else
    echo -e "${GREEN}✅ docker-credential-yc уже установлен${NC}"
fi

# Запрашиваем ID реестра
read -p "Введите ID вашего Container Registry (например, crp9tqoau5p3b0oq9g): " REGISTRY_ID
if [ -z "$REGISTRY_ID" ]; then
    echo -e "${RED}❌ ID реестра не может быть пустым${NC}"
    exit 1
fi

# Запрашиваем токен бота
read -p "Введите TELEGRAM_BOT_TOKEN: " BOT_TOKEN
if [ -z "$BOT_TOKEN" ]; then
    echo -e "${RED}❌ Токен бота не может быть пустым${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Проверки пройдены${NC}"

# 1. Сборка Docker образа
echo -e "${YELLOW}🔨 Сборка Docker образа...${NC}"
docker build -t cr.yandex/$REGISTRY_ID/telegram-bot:latest -f Dockerfile.yc .

# 2. Авторизация в Container Registry
echo -e "${YELLOW}🔑 Авторизация в Container Registry...${NC}"
if ! yc container registry configure-docker; then
    echo -e "${RED}❌ Ошибка авторизации в Container Registry${NC}"
    echo -e "${YELLOW}⚠️  Попробуйте вручную:${NC}"
    echo "yc iam create-token | docker login --username iam --password-stdin cr.yandex"
    exit 1
fi

# 3. Загрузка образа в реестр
echo -e "${YELLOW}📦 Загрузка образа в Container Registry...${NC}"
docker push cr.yandex/$REGISTRY_ID/telegram-bot:latest

# 4. Создание Serverless Container (если не существует)
echo -e "${YELLOW}🚀 Создание Serverless Container...${NC}"
if ! yc serverless container get --name telegram-bot &> /dev/null; then
    if ! yc serverless container create --name telegram-bot; then
        echo -e "${RED}❌ Ошибка создания контейнера${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ Контейнер создан${NC}"
else
    echo -e "${YELLOW}⚠️  Контейнер уже существует, обновляем...${NC}"
fi

# 5. Создание новой ревизии контейнера
echo -e "${YELLOW}⚙️  Создание новой ревизии контейнера...${NC}"
if ! yc serverless container revision deploy \
    --container-name telegram-bot \
    --image cr.yandex/$REGISTRY_ID/telegram-bot:latest \
    --cores 1 \
    --memory 128MB \
    --concurrency 1 \
    --execution-timeout 300s \
    --environment "TELEGRAM_BOT_TOKEN=$BOT_TOKEN"; then
    echo -e "${RED}❌ Ошибка деплоя ревизии${NC}"
    exit 1
fi

echo -e "${GREEN}🎉 Деплой завершён успешно!${NC}"
echo -e "${YELLOW}📋 Проверьте статус:${NC}"
yc serverless container revision list --container-name telegram-bot