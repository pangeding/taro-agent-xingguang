import random
from typing import Optional
from ..db.models import TarotCard


def get_all_cards() -> list[dict]:
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


def get_card_by_id(card_id: int) -> Optional[dict]:
    """根据 ID 获取塔罗牌详情"""
    card = TarotCard.get_or_none(TarotCard.id == card_id)
    if not card:
        return None
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


def get_random_card() -> Optional[dict]:
    """随机抽取一张塔罗牌"""
    card_ids = [card.id for card in TarotCard.select(TarotCard.id)]
    if not card_ids:
        return None

    card_id = random.choice(card_ids)
    card = TarotCard.get(TarotCard.id == card_id)
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
