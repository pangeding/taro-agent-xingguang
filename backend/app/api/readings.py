from fastapi import APIRouter, HTTPException, WebSocket, WebSocketDisconnect
from ..service import reading_service
from ..model.request import ReadingRequest, ReadingRequestV2
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


@router.post("/langgraph", response_model=ReadingResponse)
async def create_reading_langgraph(request: ReadingRequestV2):
    """创建新的占卜（LangGraph）"""
    result = await reading_service.create_reading_langgraph(
        question=request.question,
        spread_type=request.spread_type,
        session_id=request.session_id,
        model_name=request.model_name,
    )
    return ReadingResponse(**result)


@router.get("/{reading_id}", response_model=ReadingResponse)
async def get_reading(reading_id: int):
    """获取占卜详情"""
    result = reading_service.get_reading_by_id(reading_id)
    if not result:
        raise HTTPException(status_code=404, detail="占卜记录不存在")
    return ReadingResponse(**result)


@router.websocket("/ws")
async def websocket_reading(websocket: WebSocket):
    """WebSocket 占卜端"""
    await websocket.accept()
    session_id = None
    try:
        while True:
            data = await websocket.receive_json()
            question = data.get("question", "")
            spread_type = data.get("spread_type", "single")
            model_name = data.get("model_name")
            sid = data.get("session_id") or session_id

            try:
                result = await reading_service.create_reading_langgraph(
                    question=question,
                    spread_type=spread_type,
                    session_id=sid,
                    model_name=model_name,
                )
                session_id = result.get("session_id")
                await websocket.send_json(result)
            except Exception as e:
                await websocket.send_json({
                    "error": str(e),
                    "status": "error",
                })
    except WebSocketDisconnect:
        pass
