#!/usr/bin/env python3
"""
导入塔罗牌数据到数据库
"""
import sys
import os
import json

sys.path.append(os.path.join(os.path.dirname(__file__), "..", "backend"))

from backend.app.db.models import TarotCard, create_tables
from backend.app.db.base import db


def import_cards():
    """导入塔罗牌数据"""
    data_file = os.path.join(os.path.dirname(__file__), "..", "data", "tarot_cards.json")

    if not os.path.exists(data_file):
        print(f"数据文件不存在: {data_file}")
        return

    print(f"正在从 {data_file} 导入塔罗牌数据...")

    with open(data_file, 'r', encoding='utf-8') as f:
        cards_data = json.load(f)

    # 确保数据库表存在
    with db:
        if not TarotCard.table_exists():
            create_tables()

        # 清空现有数据（可选）
        # TarotCard.delete().execute()

        # 导入数据
        imported_count = 0
        for card_data in cards_data:
            # 检查是否已存在
            existing = TarotCard.get_or_none(TarotCard.id == card_data["id"])
            if existing:
                print(f"牌 {card_data['name']} (ID: {card_data['id']}) 已存在，跳过")
                continue

            # 创建新记录
            TarotCard.create(
                id=card_data["id"],
                name=card_data["name"],
                arcana_type=card_data["arcana_type"],
                suit=card_data["suit"],
                number=card_data["number"],
                meaning_upright=card_data["meaning_upright"],
                meaning_reversed=card_data["meaning_reversed"],
                keywords=card_data["keywords"],
                element=card_data["element"],
                zodiac_sign=card_data["zodiac_sign"],
                image_url=card_data["image_url"],
                description=card_data["description"],
            )
            imported_count += 1
            print(f"导入: {card_data['name']}")

        print(f"导入完成！共导入 {imported_count} 张塔罗牌。")


if __name__ == "__main__":
    import_cards()