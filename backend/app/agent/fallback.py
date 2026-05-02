def get_basic_interpretation(card, is_reversed):
    """获取基础解读（API失败时的备用方案）"""
    meaning = card.meaning_reversed if is_reversed else card.meaning_upright
    reversal_desc = "逆位" if is_reversed else "正位"

    return f"""
    【{card.name} - {reversal_desc}】

    基础含义：{meaning}

    关键词：{card.keywords}

    （注：AI深度解读暂时不可用，这是牌面的基础含义。请结合你的具体问题思考这张牌对你的启示。）
    """.strip()
