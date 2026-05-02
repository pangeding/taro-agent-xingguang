from .state import ReadingState, CardInterpretation
from .config import get_llm_config
from .llm_client import LLMClient
from .prompts import SYSTEM_PROMPT, build_card_prompt, build_synthesis_prompt
from .fallback import get_basic_interpretation


async def interpret_cards(state: ReadingState) -> ReadingState:
    config = get_llm_config(state.get("model_name"))
    client = LLMClient(config)
    interpretations = []
    for ci in state["cards_info"]:
        card = ci["card_obj"]
        prompt = build_card_prompt(card, ci["is_reversed"], state["question"],
                                   ci["position"], state["spread_type"])
        try:
            text = await client.chat_completion(
                messages=[{"role": "user", "content": prompt}], system_prompt=SYSTEM_PROMPT)
            interpretations.append({
                "card_id": ci["id"], "card_name": ci["name"],
                "is_reversed": ci["is_reversed"], "position": ci["position"],
                "interpretation": text.strip(), "status": "success"})
        except Exception:
            interpretations.append({
                "card_id": ci["id"], "card_name": ci["name"],
                "is_reversed": ci["is_reversed"], "position": ci["position"],
                "interpretation": get_basic_interpretation(card, ci["is_reversed"]),
                "status": "fallback"})
    state["interpretations"] = interpretations
    return state


async def synthesize(state: ReadingState) -> ReadingState:
    config = get_llm_config(state.get("model_name"))
    client = LLMClient(config)
    prompt = build_synthesis_prompt(state["interpretations"], state["question"])
    try:
        state["synthesis"] = (await client.chat_completion(
            messages=[{"role": "user", "content": prompt}], system_prompt=SYSTEM_PROMPT)).strip()
    except Exception as e:
        state["synthesis"] = "（综合分析暂时不可用）"
        state["status"] = "partial"
        state["error"] = str(e)
    return state
