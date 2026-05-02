from typing import Optional


SYSTEM_PROMPT = "你是一位专业、富有洞察力的塔罗牌解读师，擅长结合牌面含义和用户具体问题提供深入、个性化的解读。你的解读充满智慧、共情和启发性。"


def build_card_prompt(card, is_reversed, question, position=None, spread_type="single"):
    """构建单张牌的AI提示词"""
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


def build_synthesis_prompt(interpretations, question):
    cards_text = "\n".join([
        f"- 位置{i+1}（{'过去' if i==0 else '现在' if i==1 else '未来'}）: "
        f"{interp['card_name']}（{'逆位' if interp['is_reversed'] else '正位'}）\n"
        f"  解读: {interp['interpretation']}"
        for i, interp in enumerate(interpretations)
    ])
    return f"""
    你是一位专业的塔罗牌解读师。以下是三张牌的独立解读：

    ## 用户问题
    {question}

    ## 各牌解读
    {cards_text}

    ## 综合解读要求
    1. 将三张牌串联为一个完整的时间线叙事
    2. 分析牌与牌之间的能量流转和关联
    3. 给出整体性的建议
    4. 字数在300-500字左右
    """.strip()
