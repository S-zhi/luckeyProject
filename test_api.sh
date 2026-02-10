#!/bin/bash

set -e

BASE_URL="http://localhost:8080/v1"
TS=$(date +%s%N)

echo "=== 1. 测试创建模型 ==="
curl -sS -X POST "$BASE_URL/models" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"smoke_model_${TS}\",
    \"storage_server\": \"nas-01\",
    \"model_path\": \"/tmp/smoke_model.pt\",
    \"impl_type\": \"yolo_ultralytics\",
    \"dataset_id\": 1,
    \"size_mb\": 95.5,
    \"version\": \"v1.0.0\",
    \"task_type\": \"detect\"
  }"
echo -e "\n"

echo "=== 2. 测试创建数据集 ==="
curl -sS -X POST "$BASE_URL/datasets" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"smoke_dataset_${TS}\",
    \"storage_server\": \"nas-01\",
    \"task_type\": \"detect\",
    \"dataset_format\": \"yolo\",
    \"dataset_path\": \"/tmp/smoke_dataset\",
    \"version\": \"v1.0.0\",
    \"size_mb\": 123.456
  }"
echo -e "\n"

echo "=== 3. 测试模型查询（过滤+排序） ==="
curl -sS -X GET "$BASE_URL/models?page=1&page_size=5&impl_type=yolo_ultralytics&size_sort=desc"
echo -e "\n"

echo "=== 4. 测试数据集查询（过滤+排序） ==="
curl -sS -X GET "$BASE_URL/datasets?task_type=detect&dataset_format=yolo&size_sort=desc"
echo -e "\n"

echo "=== 5. 测试创建训练结果 ==="
curl -sS -X POST "$BASE_URL/training-results" \
  -H "Content-Type: application/json" \
  -d '{
    "model_id": 1,
    "dataset_id": 1,
    "dataset_version": 1.0,
    "training_status": 2,
    "metric_detail": {"mAP50": 0.88, "mAP50-95": 0.65},
    "weight_path": "/tmp/train_best.pt",
    "comet_log_url": "https://comet.com/exp/123"
  }'
echo -e "\n"

echo "=== 6. 测试训练结果过滤查询 ==="
curl -sS -X GET "$BASE_URL/training-results?training_status=2"
echo -e "\n"
