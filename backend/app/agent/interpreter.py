from typing import Optional
import httpx
import asyncio
from ..core.config import settings
from ..db.models import TarotCard

# 缓存API客户端
_client = None


def get_client() -> httpx.AsyncClient:
    """获取HTTP客户端"""
    global _client
    if _client is None:
        _client = httpx.AsyncClient(
            timeout=30.0,
            headers={
                "Authorization": f"Bearer {settings.DEEPSEEK_API_KEY}",
                "Content-Type": "application/json",
            },
        )
    return _client


async def interpret_card(
    card: TarotCard,
    is_reversed: bool,
    question: str,
    position: Optional[int] = None,
    spread_type: str = "single",
) -> str:
    """
    解读单张塔罗牌

    Args:
        card: 塔罗牌对象
        is_reversed: 是否逆位
        question: 用户问题
        position: 在牌阵中的位置（从0开始）
        spread_type: 牌阵类型

    Returns:
        AI解读文本
    """
    # 构建提示词
    prompt = build_prompt(card, is_reversed, question, position, spread_type)

    try:
        # 调用DeepSeek API
        response = await call_deepseek_api(prompt)
        return response.strip()
    except Exception as e:
        # 如果API调用失败，返回基础解读
        print(f"API调用失败: {e}")
        return get_basic_interpretation(card, is_reversed)


def build_prompt(
    card: TarotCard,
    is_reversed: bool,
    question: str,
    position: Optional[int],
    spread_type: str,
) -> str:
    """构建AI提示词"""
    # 位置描述
    position_desc = ""
    if spread_type == "three" and position is not None:
        positions = ["过去", "现在", "未来"]
        if position < len(positions):
            position_desc = f"这张牌在牌阵中代表{positions[position]}。"

    # 正逆位描述
    reversal_desc = "逆位" if is_reversed else "正位"
    meaning = card.meaning_reversed if is_reversed else card.meaning_upright

    prompt = f"""
    你是一位专业的塔罗牌解读师，请为以下抽牌结果提供深入、富有洞察力的解读：

    ## 用户问题
    {question}

    ## 抽牌结果
    - 牌名：{card.name}
    - 牌型：{card.arcana_type}（{"大阿尔卡纳" if card.arcana_type == "major" else "小阿尔卡纳"}）
    - 方位：{reversal_desc}
    {f"- 花色：{card.suit}" if card.suit else ""}
    {f"- 数字：{card.number}" if card.number else ""}
    - 关键词：{card.keywords}
    {position_desc}

    ## 牌面基础含义
    {meaning}

    ## 解读要求
    1. 结合用户问题分析这张牌的含义
    2. 解释这张牌在当前问题中的象征意义
    3. 提供具体的建议或启示
    4. 语言亲切、富有共情力
    5. 避免过于笼统的描述，要具体化
    6. 字数在200-300字左右

    请开始你的解读：
    """

    return prompt.strip()


async def call_deepseek_api(prompt: str) -> str:
    """调用DeepSeek API"""
    if not settings.DEEPSEEK_API_KEY:
        raise ValueError("DeepSeek API密钥未配置")

    client = get_client()
    url = f"{settings.DEEPSEEK_API_BASE}/chat/completions"

    payload = {
        "model": settings.DEEPSEEK_MODEL,
        "messages": [
            {
                "role": "system",
                "content": "你是一位专业、富有洞察力的塔罗牌解读师，擅长结合牌面含义和用户具体问题提供深入、个性化的解读。你的解读充满智慧、共情和启发性。",
            },
            {"role": "user", "content": prompt},
        ],
        "temperature": 0.7,
        "max_tokens": 1000,
    }

    response = await client.post(url, json=payload)
    response.raise_for_status()

    data = response.json()
    return data["choices"][0]["message"]["content"]


def get_basic_interpretation(card: TarotCard, is_reversed: bool) -> str:
    """获取基础解读（API失败时的备用方案）"""
    meaning = card.meaning_reversed if is_reversed else card.meaning_upright
    reversal_desc = "逆位" if is_reversed else "正位"

    return f"""
    【{card.name} - {reversal_desc}】

    基础含义：{meaning}

    关键词：{card.keywords}

    （注：AI深度解读暂时不可用，这是牌面的基础含义。请结合你的具体问题思考这张牌对你的启示。）
    """.strip()


async def close_client():
    """关闭HTTP客户端"""
    global _client
    if _client:
        await _client.aclose()
        _client = None