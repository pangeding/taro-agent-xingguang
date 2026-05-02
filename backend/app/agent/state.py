from typing import TypedDict, Optional, Literal


class CardInterpretation(TypedDict):
    card_id: int
    card_name: str
    is_reversed: bool
    position: int
    interpretation: str
    status: Literal["success", "fallback"]


class ReadingState(TypedDict):
    question: str
    spread_type: Literal["single", "three"]
    session_id: str
    model_name: Optional[str]
    cards_info: list                          # [{id, name, is_reversed, position, card_obj}]
    interpretations: list[CardInterpretation]
    synthesis: Optional[str]
    status: Literal["success", "partial", "failed"]
    error: Optional[str]
