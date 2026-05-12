from langgraph.graph import StateGraph, END
from app.agent.state import ReadingState
from app.agent.nodes import interpret_cards, synthesize


def create_reading_graph():
    builder = StateGraph(ReadingState)
    builder.add_node("interpret", interpret_cards)
    builder.add_node("synthesize", synthesize)
    builder.set_entry_point("interpret")
    builder.add_conditional_edges("interpret",
        lambda s: "synthesize" if s["spread_type"] == "three" else END)
    builder.add_edge("synthesize", END)
    return builder.compile()

reading_graph = create_reading_graph()
