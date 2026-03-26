from peewee import (
    AutoField,
    CharField,
    TextField,
    BooleanField,
    IntegerField,
    DateTimeField,
    ForeignKeyField,
)
from datetime import datetime
from .base import BaseModel


class TarotCard(BaseModel):
    """塔罗牌表"""

    id = AutoField(primary_key=True)
    name = CharField(max_length=50, unique=True, index=True)  # 牌名
    arcana_type = CharField(
        max_length=10, choices=[("major", "大阿尔卡纳"), ("minor", "小阿尔卡纳")]
    )  # 牌型
    suit = CharField(
        max_length=20,
        null=True,
        choices=[
            ("wands", "权杖"),
            ("cups", "圣杯"),
            ("swords", "宝剑"),
            ("pentacles", "星币"),
        ],
    )  # 花色（小阿尔卡纳）
    number = IntegerField(null=True)  # 数字（小阿尔卡纳）
    meaning_upright = TextField()  # 正位含义
    meaning_reversed = TextField()  # 逆位含义
    keywords = CharField(max_length=255)  # 关键词
    element = CharField(max_length=20, null=True)  # 元素
    zodiac_sign = CharField(max_length=20, null=True)  # 星座
    image_url = CharField(max_length=255, null=True)  # 图片URL
    description = TextField()  # 详细描述

    class Meta:
        table_name = "tarot_cards"


class Reading(BaseModel):
    """占卜记录表"""

    id = AutoField(primary_key=True)
    session_id = CharField(
        max_length=100, index=True
    )  # 会话ID（匿名用户用会话标识）
    question = TextField()  # 用户问题
    spread_type = CharField(max_length=50, default="single")  # 牌阵类型
    created_at = DateTimeField(default=datetime.now, index=True)

    class Meta:
        table_name = "readings"


class ReadingCard(BaseModel):
    """抽牌结果表"""

    id = AutoField(primary_key=True)
    reading = ForeignKeyField(Reading, backref="cards", on_delete="CASCADE")
    card = ForeignKeyField(TarotCard, backref="readings")
    position = IntegerField(default=0)  # 在牌阵中的位置
    is_reversed = BooleanField(default=False)  # 是否逆位
    interpretation = TextField()  # AI解读

    class Meta:
        table_name = "reading_cards"


class Feedback(BaseModel):
    """用户反馈表"""

    id = AutoField(primary_key=True)
    reading = ForeignKeyField(Reading, backref="feedback", on_delete="CASCADE")
    rating = IntegerField()  # 评分 1-5
    comment = TextField(null=True)
    created_at = DateTimeField(default=datetime.now)

    class Meta:
        table_name = "feedbacks"


# 创建所有表
def create_tables():
    """创建所有数据库表"""
    tables = [TarotCard, Reading, ReadingCard, Feedback]
    for table in tables:
        if not table.table_exists():
            table.create_table()


def drop_tables():
    """删除所有数据库表（开发用）"""
    tables = [TarotCard, Reading, ReadingCard, Feedback]
    for table in tables:
        if table.table_exists():
            table.drop_table()