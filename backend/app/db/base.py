from peewee import SqliteDatabase, Model
from ..core.config import settings

# 创建数据库连接
database = SqliteDatabase(
    settings.DATABASE_URL.replace("sqlite:///", ""),
    pragmas={
        "foreign_keys": 1,  # 启用外键约束
        "journal_mode": "wal",  # WAL模式提高并发性能
        "cache_size": -64 * 1000,  # 64MB缓存
    },
)


class BaseModel(Model):
    """基础模型类"""

    class Meta:
        database = database


# 导出数据库实例
db = database