class TestElement extends HTMLElement{
	connectedCallback(){
		const shadow=this.attachShadow({mode:'open'})
		shadow.innerHTML="<b>Test</b>"
	}
}

customElements.define('test-element',TestElement);