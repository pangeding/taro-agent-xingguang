from fastapi import APIRouter, HTTPException
from typing import List
from ...db.models import TarotCard

router = APIRouter()


@router.get("/", response_model=List[dict])
async def get_all_cards():
    """获取所有塔罗牌"""
    cards = TarotCard.select()
    return [
        {
            "id": card.id,
            "name": card.name,
            "arcana_type": card.arcana_type,
            "suit": card.suit,
            "number": card.number,
            "keywords": card.keywords,
            "image_url": card.image_url,
        }
        for card in cards
    ]


@router.get("/{card_id}", response_model=dict)
async def get_card(card_id: int):
    """获取单张塔罗牌详情"""
    card = TarotCard.get_or_none(TarotCard.id == card_id)
    if not card:
        raise HTTPException(status_code=404, detail="牌不存在")
    return {
        "id": card.id,
        "name": card.name,
        "arcana_type": card.arcana_type,
        "suit": card.suit,
        "number": card.number,
        "meaning_upright": card.meaning_upright,
        "meaning_reversed": card.meaning_reversed,
        "keywords": card.keywords,
        "element": card.element,
        "zodiac_sign": card.zodiac_sign,
        "image_url": card.image_url,
        "description": card.description,
    }


@router.get("/random/", response_model=dict)
async def get_random_card():
    """随机抽取一张塔罗牌"""
    import random

    # 获取所有牌ID
    card_ids = [card.id for card in TarotCard.select(TarotCard.id)]
    if not card_ids:
        raise HTTPException(status_code=404, detail="暂无塔罗牌数据")

    # 随机选择一张牌
    card_id = random.choice(card_ids)
    card = TarotCard.get(TarotCard.id == card_id)

    # 随机决定正逆位
    is_reversed = random.choice([True, False])

    return {
        "id": card.id,
        "name": card.name,
        "arcana_type": card.arcana_type,
        "suit": card.suit,
        "number": card.number,
        "is_reversed": is_reversed,
        "meaning": card.meaning_reversed if is_reversed else card.meaning_upright,
        "keywords": card.keywords,
        "image_url": card.image_url,
    }