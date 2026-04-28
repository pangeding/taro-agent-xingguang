from fastapi import APIRouter, HTTPException
from ..service import card_service

router = APIRouter()


@router.get("/")
async def get_all_cards():
    """获取所有塔罗牌"""
    return card_service.get_all_cards()


@router.get("/{card_id}")
async def get_card(card_id: int):
    """获取单张塔罗牌详情"""
    card = card_service.get_card_by_id(card_id)
    if not card:
        raise HTTPException(status_code=404, detail="牌不存在")
    return card


@router.get("/random/")
async def get_random_card():
    """随机抽取一张塔罗牌"""
    card = card_service.get_random_card()
    if not card:
        raise HTTPException(status_code=404, detail="暂无塔罗牌数据")
    return card
