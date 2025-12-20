# После создания/выбора SA добавляем эту проверку
echo -e "${YELLOW}🔍 Проверяем права Service Account...${NC}"

# Проверяем есть ли права на registry
if ! yc container registry list-access-bindings $REGISTRY_ID --format json | jq -r '.[] | select(.subject.id == "'$SERVICE_ACCOUNT_ID'")' | grep -q "container-registry.images.puller"; then
    echo -e "${YELLOW}⚠️  SA не имеет прав на registry, добавляем...${NC}"
    yc container registry add-access-binding $REGISTRY_ID \
        --role container-registry.images.puller \
        --subject serviceAccount:$SERVICE_ACCOUNT_ID
fi