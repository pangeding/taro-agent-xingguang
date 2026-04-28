from fastapi import APIRouter
from . import readings, cards

router = APIRouter()

# 注册各个模块的路由
router.include_router(readings.router, prefix="/readings", tags=["占卜"])
router.include_router(cards.router, prefix="/cards", tags=["塔罗牌"])