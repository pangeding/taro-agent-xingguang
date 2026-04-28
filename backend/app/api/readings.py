from fastapi import APIRouter, HTTPException
from ..service import reading_service
from ..model.request import ReadingRequest
from ..model.response import ReadingResponse

router = APIRouter()


@router.post("/", response_model=ReadingResponse)
async def create_reading(request: ReadingRequest):
    """创建新的占卜"""
    result = await reading_service.create_reading(
        question=request.question,
        spread_type=request.spread_type,
        session_id=request.session_id,
    )
    return ReadingResponse(**result)


@router.get("/{reading_id}", response_model=ReadingResponse)
async def get_reading(reading_id: int):
    """获取占卜详情"""
    result = reading_service.get_reading_by_id(reading_id)
    if not result:
        raise HTTPException(status_code=404, detail="占卜记录不存在")
    return ReadingResponse(**result)
