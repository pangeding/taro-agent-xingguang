from pydantic import BaseModel
from typing import Optional


class ReadingRequest(BaseModel):
    """占卜请求"""

    question: str
    spread_type: str = "single"  # 牌阵类型，默认为单张
    session_id: Optional[str] = None  # 会话ID，可为空
