import random
import uuid
from typing import Optional
from ..db.models import Reading, ReadingCard, TarotCard
from ..agent.interpreter import interpret_card
from ..agent import reading_graph
from ..agent.state import ReadingState


def _draw_cards(count: int) -> list[dict]:
    """随机抽取指定数量的牌"""
    card_ids = [card.id for card in TarotCard.select(TarotCard.id)]
    if not card_ids:
        raise RuntimeError("塔罗牌数据未初始化")

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


async def _generate_interpretations(reading_id: int):
    """生成AI解读"""
    try:
        reading = Reading.get(Reading.id == reading_id)
        reading_cards = ReadingCard.select().where(ReadingCard.reading == reading)

        for rc in reading_cards:
            interpretation = await interpret_card(
                card=rc.card,
                is_reversed=rc.is_reversed,
                question=reading.question,
                position=rc.position,
                spread_type=reading.spread_type,
            )
            rc.interpretation = interpretation
            rc.save()

    except Exception as e:
        print(f"生成解读失败: {e}")


async def create_reading(question: str, spread_type: str, session_id: Optional[str] = None) -> dict:
    """创建新的占卜"""
    session_id = session_id or str(uuid.uuid4())

    reading = Reading.create(
        session_id=session_id,
        question=question,
        spread_type=spread_type,
    )

    if spread_type == "single":
        cards = _draw_cards(1)
    elif spread_type == "three":
        cards = _draw_cards(3)
    else:
        cards = _draw_cards(1)

    reading_cards = []
    for i, card_data in enumerate(cards):
        reading_card = ReadingCard.create(
            reading=reading,
            card=TarotCard.get(TarotCard.id == card_data["id"]),
            position=i,
            is_reversed=card_data["is_reversed"],
            interpretation="",
        )
        reading_cards.append(reading_card)

    await _generate_interpretations(reading.id)

    reading_cards = ReadingCard.select().where(ReadingCard.reading == reading)

    return {
        "reading_id": reading.id,
        "session_id": session_id,
        "question": reading.question,
        "spread_type": reading.spread_type,
        "created_at": reading.created_at,
        "cards": [
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
    }


def get_reading_by_id(reading_id: int) -> Optional[dict]:
    """获取占卜详情"""
    reading = Reading.get_or_none(Reading.id == reading_id)
    if not reading:
        return None

    reading_cards = ReadingCard.select().where(ReadingCard.reading == reading)

    return {
        "reading_id": reading.id,
        "session_id": reading.session_id,
        "question": reading.question,
        "spread_type": reading.spread_type,
        "created_at": reading.created_at,
        "cards": [
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
    }


async def _generate_interpretations_langgraph(reading_id: int, model_name: str = None):
    """使用 LangGraph 生成AI解读"""
    reading = Reading.get(Reading.id == reading_id)
    reading_cards = list(ReadingCard.select().where(ReadingCard.reading == reading))
    initial_state = ReadingState(
        question=reading.question, spread_type=reading.spread_type,
        session_id=reading.session_id, model_name=model_name,
        cards_info=[{"id": rc.card.id, "name": rc.card.name,
                     "is_reversed": rc.is_reversed, "position": rc.position,
                     "card_obj": rc.card} for rc in reading_cards],
        interpretations=[], synthesis=None, status="success", error=None)
    result = await reading_graph.ainvoke(initial_state)
    for i, interp in enumerate(result["interpretations"]):
        reading_cards[i].interpretation = interp["interpretation"]
        reading_cards[i].save()
    if result.get("synthesis") and reading.spread_type == "three":
        last_card = reading_cards[-1]
        last_card.interpretation += f"\n\n---\n**综合解读**: {result['synthesis']}"
        last_card.save()


async def create_reading_langgraph(question: str, spread_type: str, session_id: Optional[str] = None, model_name: Optional[str] = None) -> dict:
    """创建新的占卜（使用 LangGraph）"""
    session_id = session_id or str(uuid.uuid4())

    reading = Reading.create(
        session_id=session_id,
        question=question,
        spread_type=spread_type,
    )

    if spread_type == "single":
        cards = _draw_cards(1)
    elif spread_type == "three":
        cards = _draw_cards(3)
    else:
        cards = _draw_cards(1)

    reading_cards = []
    for i, card_data in enumerate(cards):
        reading_card = ReadingCard.create(
            reading=reading,
            card=TarotCard.get(TarotCard.id == card_data["id"]),
            position=i,
            is_reversed=card_data["is_reversed"],
            interpretation="",
        )
        reading_cards.append(reading_card)

    await _generate_interpretations_langgraph(reading.id, model_name)

    reading_cards = ReadingCard.select().where(ReadingCard.reading == reading)

    return {
        "reading_id": reading.id,
        "session_id": session_id,
        "question": reading.question,
        "spread_type": reading.spread_type,
        "created_at": reading.created_at,
        "cards": [
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
    }
