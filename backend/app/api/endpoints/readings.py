from fastapi import APIRouter, HTTPException, BackgroundTasks
from pydantic import BaseModel
from typing import Optional, List
import uuid
import random
from datetime import datetime

from ...db.models import Reading, ReadingCard, TarotCard
from ...agent.interpreter import interpret_card

router = APIRouter()


class ReadingRequest(BaseModel):
    """占卜请求"""

    question: str
    spread_type: str = "single"  # 牌阵类型，默认为单张
    session_id: Optional[str] = None  # 会话ID，可为空


class ReadingResponse(BaseModel):
    """占卜响应"""

    reading_id: int
    session_id: str
    question: str
    spread_type: str
    created_at: datetime
    cards: List[dict]


@router.post("/", response_model=ReadingResponse)
async def create_reading(request: ReadingRequest, background_tasks: BackgroundTasks):
    """创建新的占卜"""
    # 生成或使用会话ID
    session_id = request.session_id or str(uuid.uuid4())

    # 创建占卜记录
    reading = Reading.create(
        session_id=session_id,
        question=request.question,
        spread_type=request.spread_type,
    )

    # 根据牌阵类型抽牌
    if request.spread_type == "single":
        # 单张牌阵：随机抽取一张牌
        cards = await draw_cards(1)
    elif request.spread_type == "three":
        # 三张牌阵：过去、现在、未来
        cards = await draw_cards(3)
    else:
        # 默认单张
        cards = await draw_cards(1)

    # 保存抽牌结果（先保存基础信息，AI解读在后台进行）
    reading_cards = []
    for i, card_data in enumerate(cards):
        reading_card = ReadingCard.create(
            reading=reading,
            card=TarotCard.get(TarotCard.id == card_data["id"]),
            position=i,
            is_reversed=card_data["is_reversed"],
            interpretation="",  # 初始为空，后台任务会填充
        )
        reading_cards.append(reading_card)

    # 后台任务：AI解读
    background_tasks.add_task(generate_interpretations, reading.id)

    # 返回响应
    return ReadingResponse(
        reading_id=reading.id,
        session_id=session_id,
        question=reading.question,
        spread_type=reading.spread_type,
        created_at=reading.created_at,
        cards=[
            {
                "card_id": rc.card.id,
                "name": rc.card.name,
                "position": rc.position,
                "is_reversed": rc.is_reversed,
                "interpretation": rc.interpretation,
                "image_url": rc.card.image_url,
            }
            for rc in reading_cards
        ],
    )


@router.get("/{reading_id}", response_model=ReadingResponse)
async def get_reading(reading_id: int):
    """获取占卜详情"""
    reading = Reading.get_or_none(Reading.id == reading_id)
    if not reading:
        raise HTTPException(status_code=404, detail="占卜记录不存在")

    # 获取关联的牌
    reading_cards = ReadingCard.select().where(ReadingCard.reading == reading)

    return ReadingResponse(
        reading_id=reading.id,
        session_id=reading.session_id,
        question=reading.question,
        spread_type=reading.spread_type,
        created_at=reading.created_at,
        cards=[
            {
                "card_id": rc.card.id,
                "name": rc.card.name,
                "position": rc.position,
                "is_reversed": rc.is_reversed,
                "interpretation": rc.interpretation,
                "image_url": rc.card.image_url,
            }
            for rc in reading_cards
        ],
    )


async def draw_cards(count: int) -> List[dict]:
    """随机抽取指定数量的牌"""
    # 获取所有牌ID
    card_ids = [card.id for card in TarotCard.select(TarotCard.id)]
    if not card_ids:
        raise HTTPException(status_code=500, detail="塔罗牌数据未初始化")

    # 随机选择不重复的牌
    selected_ids = random.sample(card_ids, min(count, len(card_ids)))
    cards = []

    for card_id in selected_ids:
        card = TarotCard.get(TarotCard.id == card_id)
        is_reversed = random.choice([True, False])
        cards.append(
            {
                "id": card.id,
                "name": card.name,
                "is_reversed": is_reversed,
            }
        )

    return cards


async def generate_interpretations(reading_id: int):
    """生成AI解读（后台任务）"""
    try:
        reading = Reading.get(Reading.id == reading_id)
        reading_cards = ReadingCard.select().where(ReadingCard.reading == reading)

        for rc in reading_cards:
            # 使用AI代理进行解读
            interpretation = await interpret_card(
                card=rc.card,
                is_reversed=rc.is_reversed,
                question=reading.question,
                position=rc.position,
                spread_type=reading.spread_type,
            )

            # 更新解读结果
            rc.interpretation = interpretation
            rc.save()

    except Exception as e:
        print(f"生成解读失败: {e}")