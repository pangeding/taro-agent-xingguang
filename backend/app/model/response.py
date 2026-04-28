from pydantic import BaseModel
from datetime import datetime
from typing import Optional, List


class CardResponse(BaseModel):
    """塔罗牌响应"""

    id: int
    name: str
    arcana_type: str
    suit: Optional[str]
    number: Optional[int]
    keywords: str
    image_url: Optional[str]


class CardDetailResponse(BaseModel):
    """塔罗牌详情响应"""

    id: int
    name: str
    arcana_type: str
    suit: Optional[str]
    number: Optional[int]
    meaning_upright: str
    meaning_reversed: str
    keywords: str
    element: Optional[str]
    zodiac_sign: Optional[str]
    image_url: Optional[str]
    description: str


class RandomCardResponse(BaseModel):
    """随机抽牌响应"""

    id: int
    name: str
    arcana_type: str
    suit: Optional[str]
    number: Optional[int]
    is_reversed: bool
    meaning: str
    keywords: str
    image_url: Optional[str]


class ReadingCardResponse(BaseModel):
    """抽牌结果响应"""

    card_id: int
    name: str
    position: int
    is_reversed: bool
    interpretation: str
    image_url: Optional[str]


class ReadingResponse(BaseModel):
    """占卜响应"""

    reading_id: int
    session_id: str
    question: str
    spread_type: str
    created_at: datetime
    cards: List[ReadingCardResponse]
