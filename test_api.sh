#!/bin/bash

BASE_URL="http://localhost:8080/v1"

echo "=== 1. 测试创建模型 ==="
curl -X POST "$BASE_URL/models" \
     -H "Content-Type: application/json" \
     -d '{
        "model_name": "ResNet50_Traffic",
        "model_type": 3,
        "model_version": 1.2,
        "algorithm": "ResNet",
        "framework": "PyTorch",
        "weight_size_mb": 95.5,
        "weight_path": "/home/user/weights/resnet50.pt"
     }'
echo -e "\n"

echo "=== 2. 测试创建数据集 ==="
curl -X POST "$BASE_URL/datasets" \
     -H "Content-Type: application/json" \
     -d '{
        "dataset_name": "Traffic_Signs_V1",
        "dataset_type": 1,
        "sample_count": 5000,
        "storage_type": 1,
        "dataset_path": "/data/datasets/traffic",
        "annotation_type": 1,
        "description": "交通标志检测数据集"
     }'
echo -e "\n"

echo "=== 3. 测试模型分页查询 + 过滤 + 排序 ==="
curl -X GET "$BASE_URL/models?page=1&page_size=5&algorithm=ResNet&weight_sort=desc"
echo -e "\n"

echo "=== 4. 测试数据集组合查询 ==="
curl -X GET "$BASE_URL/datasets?dataset_type=1&storage_type=1"
echo -e "\n"

echo "=== 5. 测试创建训练结果 ==="
curl -X POST "$BASE_URL/training-results" \
     -H "Content-Type: application/json" \
     -d '{
        "model_id": 1,
        "dataset_id": 1,
        "dataset_version": 1.0,
        "training_status": 2,
        "metric_detail": {"mAP50": 0.88, "mAP50-95": 0.65},
        "weight_path": "/path/to/train/best.pt",
        "comet_log_url": "https://comet.com/exp/123"
     }'
echo -e "\n"

echo "=== 6. 测试训练结果过滤查询 ==="
curl -X GET "$BASE_URL/training-results?training_status=2"
echo -e "\n"

echo "=== 7. 测试百度云上传 (需要 ID，这里演示路径) ==="
echo "请手动测试: curl -X POST $BASE_URL/models/1/upload"
